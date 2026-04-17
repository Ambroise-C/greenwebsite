let currentUser = null;
let state = {
    user: null,
    family: { owner: '', members: [] },
    privateTasks: [],
    familyTasks: []
};
let authToken = localStorage.getItem('leafToken') || null;

const taskInput = document.getElementById('taskInput');
const taskScope = document.getElementById('taskScope');
const addBtn = document.getElementById('addBtn');

async function api(path, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...(options.headers || {})
    };
    if (authToken) {
        headers.Authorization = `Bearer ${authToken}`;
    }

    const response = await fetch(path, {
        ...options,
        headers
    });

    let payload = null;
    try {
        payload = await response.json();
    } catch (_) {
        payload = null;
    }

    if (!response.ok) {
        const errorMessage = payload && payload.error ? payload.error : 'Erreur serveur';
        throw new Error(errorMessage);
    }

    return payload;
}

// --- AUTH ---
async function handleAuth() {
    const user = document.getElementById('userIn').value.trim();
    const pass = document.getElementById('passIn').value.trim();
    if (!user || !pass) return;

    try {
        const result = await api('/api/auth', {
            method: 'POST',
            body: JSON.stringify({ username: user, password: pass })
        });

        authToken = result.token;
        localStorage.setItem('leafToken', authToken);
        applyState(result.state);
    } catch (err) {
        alert(err.message);
    }
}

function applyState(newState) {
    state = newState;
    currentUser = state.user ? state.user.username : null;
    render();
}

async function refreshState() {
    if (!authToken) {
        state = {
            user: null,
            family: { owner: '', members: [] },
            privateTasks: [],
            familyTasks: []
        };
        currentUser = null;
        render();
        return;
    }

    try {
        const nextState = await api('/api/state', { method: 'GET' });
        applyState(nextState);
    } catch (_) {
        authToken = null;
        localStorage.removeItem('leafToken');
        state = {
            user: null,
            family: { owner: '', members: [] },
            privateTasks: [],
            familyTasks: []
        };
        currentUser = null;
        render();
    }
}

async function logout() {
    try {
        if (authToken) {
            await api('/api/logout', { method: 'POST' });
        }
    } catch (_) {
        // Ignore logout API errors and clear local session anyway.
    }

    authToken = null;
    localStorage.removeItem('leafToken');
    state = {
        user: null,
        family: { owner: '', members: [] },
        privateTasks: [],
        familyTasks: []
    };
    currentUser = null;
    render();
}

// --- LOGIQUE FAMILLE ---
async function joinFamily() {
    const code = document.getElementById('destCode').value.toUpperCase().trim();
    if (!code) return;

    try {
        await api('/api/family/join', {
            method: 'POST',
            body: JSON.stringify({ code })
        });
        await refreshState();
    } catch (err) {
        alert(err.message);
    }
}

async function leaveFamily() {
    try {
        await api('/api/family/leave', { method: 'POST' });
        await refreshState();
    } catch (err) {
        alert(err.message);
    }
}

async function kickMember(m) {
    try {
        await api('/api/family/kick', {
            method: 'POST',
            body: JSON.stringify({ username: m })
        });
        await refreshState();
    } catch (err) {
        alert(err.message);
    }
}

// --- TÂCHES ---
addBtn.onclick = async () => {
    const title = taskInput.value.trim();
    if (!currentUser || !title) return;

    try {
        await api('/api/tasks', {
            method: 'POST',
            body: JSON.stringify({ scope: taskScope.value, title })
        });
        taskInput.value = '';
        await refreshState();
    } catch (err) {
        alert(err.message);
    }
};

async function toggleTask(id, scope) {
    if (!currentUser) return;

    try {
        await api('/api/tasks/toggle', {
            method: 'POST',
            body: JSON.stringify({ id, scope })
        });
        await refreshState();
    } catch (err) {
        alert(err.message);
    }
}

async function deleteTask(id, scope) {
    if (!currentUser) return;

    try {
        await api('/api/tasks/delete', {
            method: 'POST',
            body: JSON.stringify({ id, scope })
        });
        await refreshState();
    } catch (err) {
        alert(err.message);
    }
}

// --- RENDU (SÉCURISÉ) ---
function render() {
    const privateBody = document.getElementById('privateTaskBody');
    const familyBody = document.getElementById('familyTaskBody');
    const loginForm = document.getElementById('login-form');
    const userLogged = document.getElementById('user-logged');
    const familyInfoZone = document.getElementById('family-info-zone');

    // SI DÉCONNECTÉ : On vide tout et on affiche le login
    if (!currentUser || !state.user) {
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
    
    const user = state.user;
    const famId = user.familyId;
    const family = state.family;

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
    privateBody.innerHTML = (state.privateTasks || []).map(t => renderRow(t, 'private')).join('') || '<tr><td colspan="3" style="text-align:center; color:#ccc;">Vide.</td></tr>';
    familyBody.innerHTML = (state.familyTasks || []).map(t => renderRow(t, 'family')).join('') || '<tr><td colspan="3" style="text-align:center; color:#ccc;">Vide.</td></tr>';

    document.getElementById('stats').innerText = `P: ${(state.privateTasks || []).length} / F: ${(state.familyTasks || []).length} / GROUPE: ${famId.toUpperCase()}`;
}

refreshState();