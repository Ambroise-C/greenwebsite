let db = JSON.parse(localStorage.getItem('leafDB')) || { users: {}, families: {} };
let currentUser = localStorage.getItem('leafCurrentSession') || null;

const taskInput = document.getElementById('taskInput');
const taskScope = document.getElementById('taskScope');
const addBtn = document.getElementById('addBtn');

function save() {
    localStorage.setItem('leafDB', JSON.stringify(db));
    render();
}

// --- AUTH ---
function handleAuth() {
    const user = document.getElementById('userIn').value.trim();
    const pass = document.getElementById('passIn').value.trim();
    if (!user || !pass) return;

    if (db.users[user]) {
        if (db.users[user].mdp === pass) login(user);
        else alert("MDP Incorrect");
    } else {
        const code = Math.random().toString(36).substring(2, 8).toUpperCase();
        db.users[user] = { mdp: pass, code: code, familyId: user, privateTasks: [] };
        db.families[user] = { owner: user, members: [user], tasks: [] };
        save();
        login(user);
    }
}

function login(u) { 
    currentUser = u; 
    localStorage.setItem('leafCurrentSession', u); 
    render(); 
}

function logout() { 
    currentUser = null; 
    localStorage.removeItem('leafCurrentSession'); 
    render(); 
}

// --- LOGIQUE FAMILLE ---
function joinFamily() {
    const code = document.getElementById('destCode').value.toUpperCase().trim();
    let targetOwner = Object.keys(db.users).find(u => db.users[u].code === code);

    if (!targetOwner || targetOwner === currentUser) return alert("Code invalide");

    leaveFamily(false); 
    db.users[currentUser].familyId = targetOwner;
    db.families[targetOwner].members.push(currentUser);
    save();
}

function leaveFamily(shouldSave = true) {
    const oldFam = db.users[currentUser].familyId;
    if (db.families[oldFam]) {
        db.families[oldFam].members = db.families[oldFam].members.filter(m => m !== currentUser);
    }
    db.users[currentUser].familyId = currentUser;
    if (!db.families[currentUser]) {
        db.families[currentUser] = { owner: currentUser, members: [currentUser], tasks: [] };
    } else if (!db.families[currentUser].members.includes(currentUser)) {
        db.families[currentUser].members.push(currentUser);
    }
    if (shouldSave) save();
}

function kickMember(m) {
    const famId = db.users[currentUser].familyId;
    db.users[m].familyId = m;
    db.families[m] = { owner: m, members: [m], tasks: [] };
    db.families[famId].members = db.families[famId].members.filter(mem => mem !== m);
    save();
}

// --- TÂCHES ---
addBtn.onclick = () => {
    const title = taskInput.value.trim();
    if (!currentUser || !title) return;

    if (taskScope.value === "private") {
        if (!db.users[currentUser].privateTasks) db.users[currentUser].privateTasks = [];
        db.users[currentUser].privateTasks.push({ id: Date.now(), title, completed: false });
    } else {
        const famId = db.users[currentUser].familyId;
        db.families[famId].tasks.push({ id: Date.now(), title, completed: false, completedBy: null });
    }
    taskInput.value = '';
    save();
};

function toggleTask(id, scope) {
    if (!currentUser) return;
    if (scope === 'private') {
        const t = db.users[currentUser].privateTasks.find(x => x.id === id);
        if (t) t.completed = !t.completed;
    } else {
        const famId = db.users[currentUser].familyId;
        const t = db.families[famId].tasks.find(x => x.id === id);
        if (t) {
            t.completed = !t.completed;
            t.completedBy = t.completed ? currentUser : null;
        }
    }
    save();
}

function deleteTask(id, scope) {
    if (!currentUser) return;
    if (scope === 'private') {
        db.users[currentUser].privateTasks = db.users[currentUser].privateTasks.filter(x => x.id !== id);
    } else {
        const famId = db.users[currentUser].familyId;
        db.families[famId].tasks = db.families[famId].tasks.filter(x => x.id !== id);
    }
    save();
}

// --- RENDU (SÉCURISÉ) ---
function render() {
    const privateBody = document.getElementById('privateTaskBody');
    const familyBody = document.getElementById('familyTaskBody');
    const loginForm = document.getElementById('login-form');
    const userLogged = document.getElementById('user-logged');
    const familyInfoZone = document.getElementById('family-info-zone');

    // SI DÉCONNECTÉ : On vide tout et on affiche le login
    if (!currentUser || !db.users[currentUser]) {
        loginForm.style.display = 'block';
        userLogged.style.display = 'none';
        familyInfoZone.style.display = 'none';
        
        // Vider les tableaux
        privateBody.innerHTML = '<tr><td colspan="3" style="text-align:center; color:#999;">Connectez-vous pour voir vos tâches.</td></tr>';
        familyBody.innerHTML = '<tr><td colspan="3" style="text-align:center; color:#999;">Connectez-vous pour voir les tâches familiales.</td></tr>';
        
        document.getElementById('stats').innerText = "Veuillez vous connecter.";
        return;
    }

    // SI CONNECTÉ : On affiche les données
    loginForm.style.display = 'none';
    userLogged.style.display = 'block';
    familyInfoZone.style.display = 'block';
    
    const user = db.users[currentUser];
    const famId = user.familyId;
    const family = db.families[famId];

    document.getElementById('user-display').innerText = currentUser.toUpperCase();
    document.getElementById('my-code').innerText = user.code;

    // Zone membres
    document.getElementById('family-join-zone').style.display = (famId === currentUser && family.members.length === 1) ? 'block' : 'none';
    document.getElementById('memberList').innerHTML = family.members.map(m => `
        <div style="display:flex; justify-content:space-between; border:1px solid var(--border); padding:10px; margin-bottom:5px; background:white; color:black;">
            <span style="font-size:0.8rem;">${m === family.owner ? '●' : '○'} ${m.toUpperCase()}</span>
            ${family.owner === currentUser && m !== currentUser ? `
                <button class="action-btn" onclick="kickMember('${m}')">VIRER</button>
            ` : ''}
        </div>
    `).join('');

    // Fonction pour générer une ligne
    const renderRow = (t, scope) => `
        <tr>
            <td><input type="checkbox" ${t.completed ? 'checked' : ''} onchange="toggleTask(${t.id}, '${scope}')"></td>
            <td class="${t.completed ? 'completed' : ''}">
                ${t.title}
                ${t.completed && t.completedBy ? `<br><span style="font-size:0.6rem; color:#4CAF50;">VALIDE PAR: ${t.completedBy}</span>` : ''}
            </td>
            <td class="text-right"><button class="action-btn" onclick="deleteTask(${t.id}, '${scope}')">DELETE</button></td>
        </tr>
    `;

    // Remplissage des tables
    privateBody.innerHTML = (user.privateTasks || []).map(t => renderRow(t, 'private')).join('') || '<tr><td colspan="3" style="text-align:center; color:#ccc;">Vide.</td></tr>';
    familyBody.innerHTML = family.tasks.map(t => renderRow(t, 'family')).join('') || '<tr><td colspan="3" style="text-align:center; color:#ccc;">Vide.</td></tr>';

    document.getElementById('stats').innerText = `P: ${(user.privateTasks || []).length} / F: ${family.tasks.length} / GROUPE: ${famId.toUpperCase()}`;
}

render();