// A simple service worker for PWA installation capability.
self.addEventListener('fetch', function(event) {
    // We are not adding any offline caching for this simple case.
    // The service worker is here just to make the app installable.
    event.respondWith(fetch(event.request));
});