# plucker
Discord bot that retrieves short form content from user-given links and downloads them to Discord chats.

https://github.com/user-attachments/assets/346824d3-238c-4040-809a-c412d55a6293

> video used for testing: https://youtube.com/shorts/T0Q3qJmRJpA?si=agJFAmmKgcyRUlRC

## Setup

1. Set up your discord bot and invite it to a server with *message content intent* and *send messages* permissions
2. Get token for bot and copy it to `.env` (use `.env.example` for this)
2. Clone the project
3. Run `go run main`

## Running with Docker

First, build the image: `docker build . -t plucker`

Afterwards, create and run the container:

1. For local development, use the --env-file flag:
    `docker run --env-file .env plucker`

2. For production, pass variables individually (CI/CD will manage injecting environment secrets):
    `docker run -e BOT_TOKEN="secret_token" -e MAX_FILE_SIZE_MB="<size in mb> -e " plucker`
