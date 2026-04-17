// 1. VARIABLES D'ÉTAT
let currentUser = localStorage.getItem('leafCurrentSession') || null;
let currentUserData = { family_id: "", code: "" }; 
let currentTasks = { private: [], family: [] };

// 2. FONCTION DE RÉFÉRENCEMENT DES ÉLÉMENTS (pour éviter les erreurs null)
function getEl(id) { return document.getElementById(id); }

// 3. SYNC AVEC LE BACKEND
async function refreshData() {
    if (!currentUser) return render();
    
    try {
        const response = await fetch(`/api/tasks?user=${currentUser}`);
        if (!response.ok) throw new Error("Erreur serveur");
        const data = await response.json();
        
        currentTasks.private = data.private || [];
        currentTasks.family = data.family || [];
        if (data.user) currentUserData = data.user;

        render();
    } catch (err) {
        console.error("Erreur refresh:", err);
    }
}

// 4. AUTHENTIFICATION (Appelée par onclick dans le HTML)
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
            currentUser = data.username;
            currentUserData = data; 
            localStorage.setItem('leafCurrentSession', currentUser);
            await refreshData();
        } else if (response.status === 401) {
            alert("Mot de passe incorrect pour ce pseudo !");
        } else {
            alert("Erreur lors de l'authentification");
}
    } catch (err) {
        alert("Serveur Go injoignable");
    }
}

// 5. AJOUT DE TÂCHE (L'action du bouton AJOUTER)
async function addTask() {
    const title = getEl('taskInput').value.trim();
    const scope = getEl('taskScope').value;

    console.log(currentUser)
    console.log(currentUserData.family_ID)

    if (!currentUser || !currentUserData.family_ID) {
        alert("Session non chargée, réessayez...");
        return;
    }
    if (!title) return;

    try {
        const response = await fetch('/api/tasks', {
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

// 6. DÉCONNEXION
function logout() { 
    currentUser = null; 
    localStorage.removeItem('leafCurrentSession'); 
    render(); 
}

// 7. RENDU DE L'INTERFACE
function render() {
    if (!currentUser) {
        getEl('login-form').style.display = 'block';
        getEl('user-logged').style.display = 'none';
        getEl('family-info-zone').style.display = 'none';
        getEl('privateTaskBody').innerHTML = '<tr><td colspan="3">Connectez-vous</td></tr>';
        return;
    }

    getEl('login-form').style.display = 'none';
    getEl('user-logged').style.display = 'block';
    getEl('family-info-zone').style.display = 'block';
    
    getEl('user-display').innerText = currentUser.toUpperCase();
    getEl('my-code').innerText = currentUserData.code || "";

    const renderRow = (t, scope) => `
        <tr>
            <td><input type="checkbox" ${t.completed ? 'checked' : ''}></td>
            <td class="${t.completed ? 'completed' : ''}">${t.title}</td>
            <td class="text-right"><button class="action-btn">DELETE</button></td>
        </tr>
    `;

    getEl('privateTaskBody').innerHTML = currentTasks.private.map(t => renderRow(t, 'private')).join('') || '<tr><td>Vide</td></tr>';
    getEl('familyTaskBody').innerHTML = currentTasks.family.map(t => renderRow(t, 'family')).join('') || '<tr><td>Vide</td></tr>';
    getEl('stats').innerText = `P: ${currentTasks.private.length} / F: ${currentTasks.family.length}`;
}

// 8. INITIALISATION AU CHARGEMENT
refreshData();