
let db = JSON.parse(localStorage.getItem('leafDB')) || {};
let currentUser = localStorage.getItem('leafCurrentSession') || null;
let tasks = [];

const taskInput = document.getElementById('taskInput');
const addBtn = document.getElementById('addBtn');
const taskBody = document.getElementById('taskBody');
const statsEl = document.getElementById('stats');

// --- SYSTÈME D'AUTHENTIFICATION ---
function handleAuth() {
    const user = document.getElementById('userIn').value.trim();
    const pass = document.getElementById('passIn').value.trim();

    if (!user || !pass) {
        alert("Veuillez remplir le pseudo et le mot de passe.");
        return;
    }

    if (db[user]) {
        // L'utilisateur existe déjà
        if (db[user].mdp === pass) {
            login(user);
        } else {
            alert("Mot de passe incorrect !");
        }
    } else {
        // Création automatique de compte (Sign-up)
        db[user] = { mdp: pass, tasks: [] };
        saveDB();
        login(user);
        alert("Compte créé ! Bienvenue 🌿");
    }
}

function login(user) {
    currentUser = user;
    localStorage.setItem('leafCurrentSession', user);
    // On vide les champs
    document.getElementById('userIn').value = "";
    document.getElementById('passIn').value = "";
    render();
}

function logout() {
    currentUser = null;
    localStorage.removeItem('leafCurrentSession');
    tasks = [];
    render();
}

// --- PERSISTENCE ---
function saveDB() {
    if (currentUser) {
        db[currentUser].tasks = tasks;
    }
    localStorage.setItem('leafDB', JSON.stringify(db));
    updateStats();
}

// --- AFFICHAGE & LOGIQUE DES TÂCHES ---
function render() {
    const loginForm = document.getElementById('login-form');
    const userLogged = document.getElementById('user-logged');
    
    if (currentUser && db[currentUser]) {
        loginForm.style.display = 'none';
        userLogged.style.display = 'block';
        document.getElementById('user-display').innerText = `👤 ${currentUser} `;
        tasks = db[currentUser].tasks; // On charge les tâches du profil
    } else {
        loginForm.style.display = 'block';
        userLogged.style.display = 'none';
        taskBody.innerHTML = '<tr><td colspan="3" style="text-align:center;">Connectez-vous pour gérer vos tâches.</td></tr>';
        updateStats();
        return;
    }

    taskBody.innerHTML = '';
    tasks.forEach(task => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td><input type="checkbox" ${task.completed ? 'checked' : ''} onchange="toggle(${task.id})"></td>
            <td class="${task.completed ? 'completed' : ''}">${task.title}</td>
            <td class="text-right">
                <button class="action-btn" onclick="remove(${task.id})">DELETE</button>
            </td>
        `;
        taskBody.appendChild(row);
    });
    updateStats();
}

function toggle(id) {
    tasks = tasks.map(t => t.id === id ? {...t, completed: !t.completed} : t);
    saveDB();
    render();
}

function remove(id) {
    tasks = tasks.filter(t => t.id !== id);
    saveDB();
    render();
}

addBtn.onclick = () => {
    if (!currentUser) return alert("Veuillez vous connecter !");
    if (!taskInput.value.trim()) return;
    
    tasks.push({ id: Date.now(), title: taskInput.value, completed: false });
    taskInput.value = '';
    saveDB();
    render();
};

function updateStats() {
    const total = tasks.length;
    const done = tasks.filter(t => t.completed).length;
    statsEl.innerText = `Total Tasks: ${total} | Completed: ${done} | Pending: ${total - done}`;
}

// Initialisation au chargement
render();