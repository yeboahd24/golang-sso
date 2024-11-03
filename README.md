# SSO

## Getting Started

### Prerequisites

- Go 1.18+
- PostgreSQL 13+

### Installation

1. Clone the repository:

```bash
https://github.com/yeboahd24/golang-sso.git
```

2. Change into the project directory:

```bash
cd golang-sso
```

3. Install dependencies:

```bash
go mod download
```

4. Create a new PostgreSQL database and user:

```bash
createdb sso
createuser -P sso
```

5. Update the `config/config.yml` file with your database connection details.
