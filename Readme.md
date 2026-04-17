# LeafTask (Go + SQLite)

Ce projet utilise maintenant une base de donnees persistante SQLite via un backend Go.
Les donnees ne dependent plus du cache navigateur.

## Prerequis

- Go 1.22 ou plus recent installe et disponible dans le PATH

## Lancer le projet

1. Ouvrir un terminal dans le dossier `server`
2. Installer les dependances et lancer le serveur:

```powershell
go mod tidy
go run .
```

3. Ouvrir le navigateur sur:

```text
http://localhost:8080
```

## Notes

- La base SQLite est creee automatiquement dans `server/data/leaftask.db`.
- Le front (`index.html`, `app.js`, `styles.css`) est servi par le backend Go.
- Le login conserve le comportement original: si l'utilisateur n'existe pas, il est cree automatiquement.



Execution Plan

Create Go backend + routes + SQLite connection
Add SQL migrations
Implement authentication
Implement private tasks
Implement family + family tasks
Adapt app.js to consume the API
Complete manual test with user scenario