let currentUser = localStorage.getItem('leafCurrentSession') || null;
let currentUserData = { family_ID: 0, code: "", user_ID: 0, members: [] }; 
let currentTasks = { private: [], family: [] };

function getEl(id) { return document.getElementById(id); }

// NEW FUNCTION: SIMPLE COPY WITHOUT COLOR EFFECT
async function copyCode() {
    const code = getEl('my-code').innerText;
    if (!code || code === "---" || code === "COPIED!") return;

    try {
        await navigator.clipboard.writeText(code);
        
        const codeEl = getEl('my-code');
        const originalText = currentUserData.code;
        
        codeEl.innerText = "COPIED!";
        setTimeout(() => {
            codeEl.innerText = originalText;
        }, 800);
    } catch (err) {
        console.error("Copy error:", err);
    }
}

async function refreshData() {
    if (!currentUser) return render();
    try {
        const response = await fetch(`/api/tasks?user=${encodeURIComponent(currentUser)}`);
        const data = await response.json();
        currentTasks.private = data.private || [];
        currentTasks.family = data.family || [];
        if (data.user) {
            currentUserData = data.user;
            if (data.family_info) {
                currentUserData.code = data.family_info.code;
                currentUserData.members = data.family_info.members;
            }
        }
        render();
    } catch (err) { console.error(err); }
}

async function toggleTask(id, isChecked) {
    await fetch(`/api/tasks?id=${id}&user=${encodeURIComponent(currentUser)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
            completed: isChecked, 
            completedBy: isChecked ? currentUser : "" 
        })
    });
    refreshData();
}

async function joinFamily() {
    const code = getEl('destCode').value.trim().toUpperCase();
    const res = await fetch(`/api/join-family?user=${encodeURIComponent(currentUser)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code })
    });
    if(res.ok) { getEl('destCode').value = ''; refreshData(); }
    else alert("Invalid code");
}

async function leaveFamily() {
    if(!confirm("Leave the family?")) return;
    await fetch(`/api/leave-family?user=${encodeURIComponent(currentUser)}`, { method: 'POST' });
    refreshData();
}

async function handleAuth() {
    const userIn = getEl('userIn');
    const passIn = getEl('passIn');
    const user = userIn.value.trim();
    const pass = passIn.value.trim();
    
    if (!user || !pass) return;

    try {
        const response = await fetch('/api/auth', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ user, pass })
        });

        if (response.ok) {
            const data = await response.json();
            // Le serveur renvoie { "username": "...", "user_ID": ..., "family_ID": ... }
            currentUser = data.username;
            currentUserData = data; 
            
            localStorage.setItem('leafCurrentSession', currentUser);
            
            // Nettoyage des champs de saisie par sécurité
            userIn.value = "";
            passIn.value = "";

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

async function addTask() {
    const title = getEl('taskInput').value.trim();
    const scope = getEl('taskScope').value;
    if (!title) return;
    
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
        console.error("Add error:", err);
    }
}

async function deleteTask(id) {
    if(!confirm("Delete this task?")) return;
    const url = `/api/tasks?id=${id}&user=${encodeURIComponent(currentUser)}`;
    try {
        const response = await fetch(url, { method: 'DELETE' });
        if (response.ok) { await refreshData(); }
    } catch (err) { console.error("Error:", err); }
}

// UPDATE: CLEAR EVERYTHING ON LOGOUT
function logout() { 
    currentUser = null; 
    currentUserData = { family_ID: 0, code: "", user_ID: 0, members: [] };
    currentTasks = { private: [], family: [] };
    localStorage.removeItem('leafCurrentSession'); 
    const memberListEl = getEl('memberList');
    if (memberListEl) memberListEl.innerText = "";
    
    render(); 
}

function render() {
    if (!currentUser) {
        getEl('login-form').style.display = 'block';
        getEl('user-logged').style.display = 'none';
        getEl('family-join-zone').style.display = 'none';
        getEl('family-info-zone').style.display = 'none';
        getEl('privateTaskBody').innerHTML = '<tr><td colspan="3">Please log in</td></tr>';
        getEl('familyTaskBody').innerHTML = '<tr><td colspan="3">Please log in</td></tr>';
        getEl('stats').innerText = "Disconnected";
        return;
    }
    getEl('login-form').style.display = 'none';
    getEl('user-logged').style.display = 'block';
    getEl('user-display').innerText = currentUser;
    const codeEl = getEl('my-code');
    codeEl.innerText = currentUserData.code || "---";
    codeEl.onclick = copyCode; 
    codeEl.style.cursor = "pointer";
    codeEl.title = "Click to copy";
    const hasFamily = currentUserData.members && currentUserData.members.length > 1;
    
    if (hasFamily) {
        getEl('family-join-zone').style.display = 'none';
        getEl('family-info-zone').style.display = 'block';
        getEl('memberList').innerText = currentUserData.members.join(', ');
    } else {
        getEl('family-join-zone').style.display = 'block';
        getEl('family-info-zone').style.display = 'none';
        getEl('memberList').innerText = "";
    }

    const renderRow = (t) => `
        <tr>
            <td>
                <input type="checkbox" ${t.completed ? 'checked' : ''} 
                    onchange="toggleTask(${t.task_ID}, this.checked)">
            </td>
            <td>
                <span class="${t.completed ? 'completed' : ''}">${t.title}</span>
                ${t.completed && t.completedBy ? `<br><small style="color:gray">✅ by ${t.completedBy}</small>` : ''}
            </td>
            <td class="text-right">
                <button class="action-btn" onclick="deleteTask(${t.task_ID})">DELETE</button>
            </td>
        </tr>
    `;

    getEl('privateTaskBody').innerHTML = currentTasks.private.map(renderRow).join('') || '<tr><td colspan="3">Empty</td></tr>';
    getEl('familyTaskBody').innerHTML = currentTasks.family.map(renderRow).join('') || '<tr><td colspan="3">Empty</td></tr>';
    
    getEl('stats').innerText = `Private: ${currentTasks.private.length} | Family: ${currentTasks.family.length}`;
}

refreshData();