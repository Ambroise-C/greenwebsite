// 1. CONFIGURATION
const SB_URL = 'https://vvxtigiomfukjrloqkcj.supabase.co';
const SB_KEY = 'sb_secret_NFYzxW4QBioyFbnnZ95k2g_3LPPfFLq';
const supabase = libsupabase.createClient(SB_URL, SB_KEY);

// DOM Elements
const authSection = document.getElementById('auth-section');
const dashboardSection = document.getElementById('dashboard-section');
const taskBody = document.getElementById('taskBody');
const taskInput = document.getElementById('taskInput');

// 2. AUTHENTICATION LOGIC
document.getElementById('loginBtn').onclick = async () => {
    const email = document.getElementById('emailInput').value;
    const password = document.getElementById('passwordInput').value;
    const { error } = await supabase.auth.signInWithPassword({ email, password });
    if (error) alert(error.message);
};

document.getElementById('signupBtn').onclick = async () => {
    const email = document.getElementById('emailInput').value;
    const password = document.getElementById('passwordInput').value;

    if (!email || !password) {
        alert("Please enter both email and password.");
        return;
    }

    console.log("Attempting sign up for:", email); // Look at your F12 Console for this!

    const { data, error } = await sb.auth.signUp({
        email: email,
        password: password,
    });

    if (error) {
        console.error("Sign up error:", error.message);
        alert("Error: " + error.message);
    } else {
        alert("Success! Check your email or try logging in if you disabled 'Confirm Email'.");
    }
};

document.getElementById('logoutBtn').onclick = async () => {
    await supabase.auth.signOut();
};

// Listen for Auth Changes (Login/Logout)
supabase.auth.onAuthStateChange((event, session) => {
    if (session) {
        authSection.style.display = 'none';
        dashboardSection.style.display = 'block';
        document.getElementById('userEmail').innerText = session.user.email;
        fetchTasks();
    } else {
        authSection.style.display = 'block';
        dashboardSection.style.display = 'none';
        taskBody.innerHTML = '';
    }
});

// 3. TASK LOGIC (talking to PostgreSQL)
async function fetchTasks() {
    const { data, error } = await supabase
        .from('tasks')
        .select('*')
        .order('id', { ascending: false });
    
    if (!error) render(data);
}

async function render(tasks) {
    taskBody.innerHTML = '';
    tasks.forEach(task => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td><input type="checkbox" ${task.is_completed ? 'checked' : ''} onchange="toggle(${task.id}, ${task.is_completed})"></td>
            <td class="${task.is_completed ? 'completed' : ''}">${task.title}</td>
            <td class="text-right">
                <button class="action-btn" onclick="remove(${task.id})">DELETE</button>
            </td>
        `;
        taskBody.appendChild(row);
    });
    updateStats(tasks);
}

document.getElementById('addBtn').onclick = async () => {
    const title = taskInput.value.trim();
    if (!title) return;
    const { error } = await supabase.from('tasks').insert([{ title }]);
    if (!error) {
        taskInput.value = '';
        fetchTasks();
    }
};

async function toggle(id, currentStatus) {
    await supabase.from('tasks').update({ is_completed: !currentStatus }).eq('id', id);
    fetchTasks();
}

async function remove(id) {
    await supabase.from('tasks').delete().eq('id', id);
    fetchTasks();
}

function updateStats(tasks) {
    const total = tasks.length;
    const done = tasks.filter(t => t.is_completed).length;
    document.getElementById('stats').innerText = `Total Tasks: ${total} | Completed: ${done} | Pending: ${total - done}`;
}

/* postgresql://postgres:[rHaG9j4n86uZQW6i]@db.vvxtigiomfukjrloqkcj.supabase.co:5432/postgres */