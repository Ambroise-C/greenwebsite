const taskInput = document.getElementById('taskInput');
const addBtn = document.getElementById('addBtn');
const taskBody = document.getElementById('taskBody');
const statsEl = document.getElementById('stats');

let tasks = [
    { id: 1, title: "Review requirements", completed: false },
    { id: 2, title: "Pure HTML optimization", completed: true }
];

function render() {
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
    render();
}

function remove(id) {
    tasks = tasks.filter(t => t.id !== id);
    render();
}

addBtn.onclick = () => {
    if (!taskInput.value.trim()) return;
    tasks.push({ id: Date.now(), title: taskInput.value, completed: false });
    taskInput.value = '';
    render();
};

function updateStats() {
    const total = tasks.length;
    const done = tasks.filter(t => t.completed).length;
    statsEl.innerText = `Total Tasks: ${total} | Completed: ${done} | Pending: ${total - done}`;
}

render();