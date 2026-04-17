package main

import (
    "log"
    "net/http"
    "mon-projet/api"
    "mon-projet/internal"
    "github.com/joho/godotenv"
)

func main() {
    // 1. Charger les variables d'environnement
    if err := godotenv.Load(); err != nil {
        log.Println("Note: Fichier .env non trouvé, utilisation des variables système")
    }

    // 2. Initialiser le client Supabase
    client := internal.InitSupabase()

    // 3. TEST DE CONNEXION (Avant de lancer le serveur)
    var testData []map[string]interface{}
    err := client.DB.From("profiles").Select("*").Limit(1).Execute(&testData)

    if err != nil {
        log.Printf("❌ ERREUR : Impossible de lire la ligne : %v", err)
    } else if len(testData) > 0 {
        log.Println("✅ LIGNE RÉCUPÉRÉE :")
        // %+v affiche les clés et les valeurs de la map
        log.Printf("%+v", testData[0]) 
    } else {
        log.Println("✅ CONNEXION OK : Mais la table est vide (aucune ligne à afficher).")
    }

    // 4. Configurer les handlers
    h := &api.Handler{SB: client}

	http.HandleFunc("/api/tasks", h.TasksHandler)
	http.HandleFunc("/api/auth", h.Auth)

    // Servir le dossier public (ton HTML/CSS/JS)
    fs := http.FileServer(http.Dir("./public"))
    http.Handle("/", fs)

    // Routes API


    // 5. LANCER LE SERVEUR (Cette ligne doit être la dernière)
    log.Println("🚀 Serveur prêt sur http://localhost:8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal("Erreur lors du lancement du serveur:", err)
    }
}