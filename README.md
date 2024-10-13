<br />
<div align="center">
<h2 align="center">Go & Next.js Blog Project</h2>

  <p align="center">
    full-stack blog application, consisting of a Go-based backend and a Next.js frontend
</p>
</div>

<!-- ABOUT THE PROJECT -->
## About The Project

This project is a full-stack blog application with a Go-based backend and a Next.js frontend, designed to provide a seamless blogging experience. It leverages Neo4j, Redis, and ScyllaDB for data storage and management, all of which are containerized using Docker for easy setup and deployment.

### Overview
- **Backend (Go)**: Handles the core functionalities for blog management, including creating and managing posts, threads, and comments. It integrates with various databases to store different types of data:
   - **Neo4j**: Used for managing relationships between blog entities, such as posts, threads and tags.
  - **Redis**: Acts as a caching layer to boost performance for accessing posts content saved in markdown files.
  - **ScyllaDB**: Used for logging and analytics, storing logs and tracking user activity.
  - **Frontend (Next.js)**: Provides a modern and responsive web interface for users to interact with the blog. It fetches data from the backend and presents it in a user-friendly format, allowing users to read posts, browse threads, and leave comments. 
- **Post Content Storage**: Blog post contents are stored as .md (Markdown) files in an S3-compatible storage solution. When running locally, **MinIO** is used as the S3 bucket implementation, making it easy to manage and test file uploads and downloads.

The creation and management of posts and threads are exposed through Swagger.

<!-- GETTING STARTED -->
## Getting Started

### Prerequisites

To run this project, you need to have the following programs installed:

- Go (version 1.23+)
- Node.js (version 22.9+)
- Docker Compose (latest version)
- Make (optional but recommended for using the provided Makefile)

### Usage
1. Clone the repo
   ```sh
   git clone https://github.com/mmich-pl/ndb
   ```
2. Copy .env.example file to .env
   ```sh
   cp .env.example .env
   ```
3. Install NPM packages and go mod tidy
   ```sh
   npm install
   go mod tidy
   ```
4. Build and Start the Services
   ```sh
   make up
   make run
   ```

   This will:
   - Start the Docker containers for Neo4j, Redis, and ScyllaDB.
   - Build and run the Go backend and the Next.js frontend.

<!-- USAGE EXAMPLES -->
## Swagger API Documentation
The Go backend includes Swagger API documentation, which provides an interactive interface for exploring and testing the available REST endpoints.

### Accessing Swagger
Once the backend is running, you can access the Swagger UI at http://localhost:8080/swagger/index.html

This interface allows you to:
- View all available endpoints and their details. 
- Execute requests and see the responses directly in your browser. 
- Explore the API without needing an external client. 
- Updating Swagger Documentation

### Update the Swagger documentation

Make sure you have annotated your Go handlers with Swagger-compatible comments.
Use a tool like swaggo to generate the updated Swagger documentation.
Install the swag CLI tool if you haven’t already:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Run the following command (in folder with `main.go` file)to generate the Swagger docs:

```bash
swag init -g ./backend/main.go
```

The swag init command will scan the Go code for Swagger annotations and update the docs directory with the latest API documentation.
