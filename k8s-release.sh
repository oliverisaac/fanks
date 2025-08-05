#!/usr/bin/env bash

set -eEuo pipefail

if git status --porcelain | grep -q .; then
  echo >&2 "Git is dirty, commit before releasing..."
  exit 1
fi

IMAGE_NAME="$1"

function echo_do() {
  {
    printf "%q " "${@}"
    echo
  } >&2
  "${@}"
}

BUILDKIT_PROGRESS=plain docker buildx build --build-arg VERSION_TAG="$(git rev-parse --short HEAD)" --platform linux/amd64 --target release --push -t "${IMAGE_NAME}" .

base_image_name="${IMAGE_NAME%:*}"

image_hash=$(docker inspect "${IMAGE_NAME}" | jq -r '.[].RepoDigests[]' | grep sha256 | head -n 1)
image_hash="@sha256${image_hash#*@sha256}"

resources=$(kubectl get sts,deployments --all-namespaces -o jsonpath='{range .items[*]}{.metadata.namespace} {.kind}/{.metadata.name}{" "}{.spec.template.spec.containers[*].image}{"\n"}{end}')

echo "$resources" |
  while read namespace resource image; do
    if [[ $image == ${IMAGE_NAME}* ]]; then
      deployed=false
      for ((i = 0; i < 60; i++)); do
        echo_do kubectl rollout restart -n "$namespace" "$resource"
        if ! echo_do kubectl get -n $namespace pods -o yaml | grep -qFe "$image_hash"; then
          echo "Unable to find $image_hash in manifest. Sleeping for 3 and trying again..."
          sleep 3
        else
          deployed=true
          echo "Deployment has updated image hash: ${image_hash}"
          break
        fi
      done
      if ! $deployed; then
        echo "Failed to deploy!"
        exit 1
      fi
    fi
  done

echo "$resources" |
  while read namespace resource image; do
    if [[ $image == $IMAGE_NAME ]]; then
      echo_do kubectl rollout status -n "$namespace" "$resource"
    fi
  done
