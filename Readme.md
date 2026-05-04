# LeafTask

LeafTask is a lightweight task management app focused on eco-responsible habits.
It is built with Go, Vanilla JS, and Supabase to keep the stack simple and fast.
The goal is to keep the product useful while staying low complexity and low energy.

Deployed URL: https://greenwebsite-production.up.railway.app/

## Team

- Rayan Boumedine - Backend and Supabase integration.
- Alexis Launay - Front-end interface and user experience.
- Ambroise Couturier - Data model and family/task flows.
- Paul Mahaut - Project coordination and documentation.

## Tech Stack

- Go - Minimal runtime overhead and a small server footprint for low energy use.
- net/http - Standard library only, so no extra framework cost or dependency weight.
- Vanilla JS - No bundler or framework, which keeps the front-end light and fast.
- HTML5/CSS3 - Native browser rendering with no extra client-side runtime.
- Supabase - Managed PostgreSQL backend reduces infrastructure overhead.
- bcrypt - Safer password storage with a single, focused cryptographic dependency.

## Local Run

Clone the repository:

```bash
git clone https://github.com/Ambroise-C/greenwebsite.git
cd greenwebsite
```

Install dependencies and run locally:

```bash
go run main.go
```

Open the app at http://localhost:8080.

## Repository Tree

```text
greenwebsite/                  # Project root
├── api/                       # HTTP handlers
├── database/schema.sql        # Supabase schema for users, families, and tasks
├── internal/                  # Supabase client and data structs
├── public/                    # Static HTML, CSS, and JS
├── main.go                    # Server entry point
├── go.mod                     # Go module and dependencies
└── Readme.md                  # Project documentation
```

## Deliverable

- [LeafTask Deliverable](docs/LeafTask_Deliverable.pdf)
