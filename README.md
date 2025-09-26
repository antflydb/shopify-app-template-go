## Template description

This is a Shopify app template written in Golang, that bootstraps the app building process. It includes:

- The setup of the client and server parts, built on top of the App Bridge.
- Shopify app installation logic.
- Examples of using the Shopify API include creating store products and counting the number of products.

## Template usage

### Prerequisites

Please ensure that the following software is installed on your computer:

- [Node.js](https://nodejs.org/)
- [Golang](https://go.dev/)
- [Docker](https://www.docker.com/)

### Getting started

1. Clone the template using the following terminal command:

    ```
    git clone https://github.com/antflydb/shopify-app-template-go.git my-shopify-app
    cd my-shopify-app
    ```

2. Install NPM dependencies:

    ```
    npm i
    ```

3. Choose your database option:

   **Option A: PostgreSQL with Docker (default)**

   Start a new Postgres container using the configuration in the **`.local.env`** file:

    ```
    docker-compose --env-file .local.env up --build postgresdb
    ```

   **Option B: SQLite (no Docker required)**

   Simply set the database type to SQLite. No additional setup needed:

    ```
    export DATABASE_TYPE=sqlite
    ```

   Or create a `.local.env` file and add:
    ```
    DATABASE_TYPE=sqlite
    SQLITE_PATH=./app.db
    ```

4. Run the project:

    **With PostgreSQL:**
    ```
    npm run dev
    ```

    **With SQLite:**
    ```
    DATABASE_TYPE=sqlite npm run dev
    ```
