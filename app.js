// --- INITIALISATION DES DONNÉES ---
let db = JSON.parse(localStorage.getItem('leafDB')) || {};
let currentUser = localStorage.getItem('leafCurrentSession') || null;
let tasks = [];

const taskInput = document.getElementById('taskInput');
const addBtn = document.getElementById('addBtn');
const taskBody = document.getElementById('taskBody');
const statsEl = document.getElementById('stats');

// --- GÉNÉRATEUR DE CODE ALÉATOIRE (6 LETTRES) ---
function generateRandomCode() {
    const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
    let result = "";
    for (let i = 0; i < 6; i++) {
        result += letters.charAt(Math.floor(Math.random() * letters.length));
    }
    return result;
}

// --- AUTHENTIFICATION ---
function handleAuth() {
    const user = document.getElementById('userIn').value.trim();
    const pass = document.getElementById('passIn').value.trim();

    if (!user || !pass) return alert("Remplis le pseudo et le mot de passe !");

    if (db[user]) {
        if (db[user].mdp === pass) {
            login(user);
        } else {
            alert("Mot de passe incorrect !");
        }
    } else {
        // Création avec génération de code unique
        db[user] = { 
            mdp: pass, 
            tasks: [], 
            code: generateRandomCode() 
        };
        saveDB();
        login(user);
        alert(`Compte créé ! Ton code de partage est : ${db[user].code}`);
    }
}

function login(user) {
    currentUser = user;
    localStorage.setItem('leafCurrentSession', user);
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

// --- ENVOYER UNE TÂCHE À UN AMI ---
function sendTaskToFriend() {
    const code = document.getElementById('destCode').value.toUpperCase().trim();
    const text = document.getElementById('destTask').value.trim();

    if (!currentUser) return alert("Connecte-toi d'abord !");
    if (code.length !== 6 || !text) return alert("Code invalide ou message vide.");

    // On cherche l'utilisateur par son code
    let foundUser = null;
    for (let username in db) {
        if (db[username].code === code) {
            foundUser = username;
            break;
        }
    }

    if (foundUser) {
        db[foundUser].tasks.push({
            id: Date.now(),
            title: `📩 FROM ${currentUser}: ${text}`,
            completed: false
        });
        localStorage.setItem('leafDB', JSON.stringify(db)); // On sauve direct dans la DB globale
        alert(`Succès ! Tâche envoyée à ${foundUser}.`);
        document.getElementById('destCode').value = "";
        document.getElementById('destTask').value = "";
    } else {
        alert("Aucun utilisateur trouvé avec ce code.");
    }
}

// --- PERSISTENCE ---
function saveDB() {
    if (currentUser) {
        db[currentUser].tasks = tasks;
    }
    localStorage.setItem('leafDB', JSON.stringify(db));
    updateStats();
}

// --- AFFICHAGE ---
function render() {
    const loginForm = document.getElementById('login-form');
    const userLogged = document.getElementById('user-logged');
    
    if (currentUser && db[currentUser]) {
        loginForm.style.display = 'none';
        userLogged.style.display = 'block';
        document.getElementById('user-display').innerText = `👤 ${currentUser}`;
        document.getElementById('my-code').innerText = `CODE: ${db[currentUser].code}`;
        tasks = db[currentUser].tasks;
    } else {
        loginForm.style.display = 'block';
        userLogged.style.display = 'none';
        taskBody.innerHTML = '<tr><td colspan="3" style="text-align:center;">Connectez-vous pour commencer.</td></tr>';
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
    if (!currentUser) return alert("Connecte-toi !");
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

render();