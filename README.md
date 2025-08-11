# <img src='static/icon.png' style='width:auth; height: 1em;' /> Fanks

Fanks is a self-hosted application that provides a space for you to write down what you are thankful for each day. It's a simple, private, and personal journal for gratitude.

## Getting Started

### Prerequisites

- Docker
- Make
- Go

### Installation

1.  Clone the repository:
    ```sh
    git clone https://github.com/your-username/fanks.git
    ```
2.  Navigate to the project directory:
    ```sh
    cd fanks
    ```
3.  Build the Docker image:
    ```sh
    make docker-build
    ```
4.  Run the Docker container:
    ```sh
    make docker-run
    ```

The application will be available at `http://localhost:8080`.

## License

This project is licensed under the AGPLv3 License - see the [LICENSE](LICENSE) file for details.

