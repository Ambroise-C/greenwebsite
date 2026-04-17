// 1. VARIABLES D'ÉTAT
let currentUser = localStorage.getItem('leafCurrentSession') || null;
let currentUserData = { family_ID: 0, code: "", user_ID: 0 }; 
let currentTasks = { private: [], family: [] };

function getEl(id) { return document.getElementById(id); }

// 3. SYNC AVEC LE BACKEND
async function refreshData() {
    if (!currentUser) return render();
    
    try {
        const response = await fetch(`/api/tasks?user=${encodeURIComponent(currentUser)}`);
        if (!response.ok) throw new Error("Erreur serveur");
        const data = await response.json();
        
        // IMPORTANT : Go renvoie { "user": {...}, "private": [...], "family": [...] }
        currentTasks.private = data.private || [];
        currentTasks.family = data.family || [];
        if (data.user) {
            currentUserData = data.user;
            // On s'assure que currentUser reste synchronisé avec le username de la DB
            currentUser = data.user.username;
        }

        render();
    } catch (err) {
        console.error("Erreur refresh:", err);
    }
}

// 4. AUTHENTIFICATION
async function handleAuth() {
    const user = getEl('userIn').value.trim();
    const pass = getEl('passIn').value.trim();
    if (!user || !pass) return;

    try {
        const response = await fetch('/api/auth', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ user, pass })
        });

        if (response.ok) {
            const data = await response.json();
            // Le serveur Go renvoie directement l'objet User ici
            currentUser = data.username;
            currentUserData = data; 
            localStorage.setItem('leafCurrentSession', currentUser);
            await refreshData();
        } else if (response.status === 401) {
            alert("Mot de passe incorrect !");
        } else {
            alert("Erreur d'authentification");
        }
    } catch (err) {
        alert("Serveur injoignable");
    }
}

// 5. AJOUT DE TÂCHE
async function addTask() {
    const title = getEl('taskInput').value.trim();
    const scope = getEl('taskScope').value;
    if (!title) return;
    const payload = {
        title: title,
        scope: scope,
        user_ID: parseInt(currentUserData.user_ID), // On force en nombre
        family_ID: parseInt(currentUserData.family_ID || 0), // On force en nombre (0 par défaut)
        completed: false
    };

    console.log("Envoi de la tâche :", payload);
    try {
        const response = await fetch(`/api/tasks?user=${encodeURIComponent(currentUser)}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                title: title,
                scope: scope,
                user_ID: currentUserData.user_ID,     
                family_ID: currentUserData.family_ID, 
                completed: false
            })
        });

        if (response.ok) {
            getEl('taskInput').value = '';
            await refreshData();
        }
    } catch (err) {
        console.error("Erreur ajout:", err);
    }
}

function logout() { 
    currentUser = null; 
    localStorage.removeItem('leafCurrentSession'); 
    render(); 
}

// 7. RENDU
function render() {
    if (!currentUser) {
        getEl('login-form').style.display = 'block';
        getEl('user-logged').style.display = 'none';
        getEl('family-info-zone').style.display = 'none';
        getEl('privateTaskBody').innerHTML = '<tr><td colspan="3">Connectez-vous</td></tr>';
        getEl('familyTaskBody').innerHTML = '<tr><td colspan="3">Connectez-vous</td></tr>';
        return;
    }

    getEl('login-form').style.display = 'none';
    getEl('user-logged').style.display = 'block';
    getEl('user-display').innerText = currentUser.toUpperCase();
    getEl('my-code').innerText = currentUserData.code || "---";

    const renderRow = (t) => `
        <tr>
            <td><input type="checkbox" ${t.completed ? 'checked' : ''}></td>
            <td class="${t.completed ? 'completed' : ''}">${t.title}</td>
            <td class="text-right"><button class="action-btn">DELETE</button></td>
        </tr>
    `;

    getEl('privateTaskBody').innerHTML = currentTasks.private.map(renderRow).join('') || '<tr><td colspan="3">Aucune tâche perso</td></tr>';
    getEl('familyTaskBody').innerHTML = currentTasks.family.map(renderRow).join('') || '<tr><td colspan="3">Aucune tâche famille</td></tr>';
    getEl('stats').innerText = `Privé: ${currentTasks.private.length} | Famille: ${currentTasks.family.length}`;
}

refreshData();