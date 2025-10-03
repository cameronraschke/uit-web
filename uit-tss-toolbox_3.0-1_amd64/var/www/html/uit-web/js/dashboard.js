let dashboardPollController = null;
let updatingNote = false;

function startDashboardPolling(intervalMs = 3000) {
  if (dashboardPollController) return dashboardPollController.stop;

  let stopped = false;
  const abortSignal = new AbortController();
  dashboardPollController = { stop };

  (async function loop() {
    while (!stopped) {
      const cycleStart = Date.now();
      try {
        await fetchDashboardData(abortSignal.signal);
      } catch (e) {
        console.error("dashboard cycle error:", e);
      }
      if (stopped) break;
      const elapsed = Date.now() - cycleStart;
      const wait = Math.max(0, intervalMs - elapsed);
      await new Promise(r => setTimeout(r, wait));
    }
    dashboardPollController = null;
  })();

  function stop() {
    if (stopped) return;
    stopped = true;
    abortSignal.abort();
  }

  return stop;
}

document.addEventListener("DOMContentLoaded", () => {
  startDashboardPolling(3000);
  fetchNotes();
  const textArea = document.getElementById('note-text');
  const noteSubmitButton = document.getElementById('update-note-button');
  noteSubmitButton.addEventListener('click', async () => {
    textArea.disabled = true;
    noteSubmitButton.disabled = true;
    if (updatingNote) return;
    updatingNote = true;
    try {
      await postNote();
    } finally {
      updatingNote = false;
      textArea.disabled = false;
      noteSubmitButton.disabled = false;
    }
  });
});

async function fetchDashboardData(signal) {
  const tasks = [
    fetchInventoryOverview(signal), 
    fetchJobQueueOverview(signal)
  ];
  const results = await Promise.allSettled(tasks);
  for (const r of results) {
    if (r.status === 'rejected') console.error('Task failed:', r.reason);
  }
}

async function fetchNotes(signal) {
  try {
    const response = await fetchData('/api/notes?note_type=general', true, { signal });
    if (!response || response.length === 0) throw new Error('No data received from /api/notes');
    const jsonParsed = JSON.parse(response);
    if (!jsonParsed || Object.keys(jsonParsed).length === 0 || (jsonParsed && typeof jsonParsed === 'object' && Object.prototype.hasOwnProperty.call(jsonParsed, '__proto__'))) {
      throw new Error('Response JSON is empty or invalid');
    }
    const noteTextArea = document.getElementById('note-text');
    const noteTime = document.getElementById('note-date');
    if (!noteTime) throw new Error('Note date element not found in DOM');
    noteTime.innerHTML = jsonParsed.time ? new Date(jsonParsed.time).toLocaleString() : 'Never';
    if (!noteTextArea) throw new Error('Note text area not found in DOM');
    noteTextArea.innerHTML = jsonParsed.note || '';
    return jsonParsed;
  } catch (err) {
    if (err.name !== 'AbortError') console.error("fetchNotes error:", err);
    return null;
  }
}

async function fetchInventoryOverview(signal) {
  try {
    const response = await fetchData('/api/dashboard/inventory_summary', true, { signal });
    if (!response || response.length === 0) throw new Error('No data received from /api/dashboard/inventory_summary');
    const jsonParsed = JSON.parse(response);
    if (!jsonParsed || Object.keys(jsonParsed).length === 0 || (jsonParsed && typeof jsonParsed === 'object' && Object.prototype.hasOwnProperty.call(jsonParsed, '__proto__'))) {
      throw new Error('Response JSON is empty or invalid');
    }
    const inventoryTableBody = document.getElementById('inventory-summary-row');
    if (!inventoryTableBody) throw new Error('Inventory table body element not found in DOM');

    const rows = Array.isArray(jsonParsed) ? jsonParsed : [jsonParsed];

    const fragment = document.createDocumentFragment();
    for (const item of rows) {
      const row = document.createElement('div');
      row.classList.add('grid-item', 'row');

      const modelCell = document.createElement('div');
      modelCell.textContent = item.system_model || 'N/A';
      modelCell.classList.add('grid-item');
      row.appendChild(modelCell);

      const countCell = document.createElement('div');
      countCell.textContent = item.system_model_count != null ? item.system_model_count : '0';
      countCell.classList.add('grid-item');
      row.appendChild(countCell);

      const checkedOutCell = document.createElement('div');
      checkedOutCell.textContent = item.total_checked_out != null ? item.total_checked_out : '0';
      checkedOutCell.classList.add('grid-item');
      row.appendChild(checkedOutCell);

      const availableCell = document.createElement('div');
      availableCell.textContent = item.available_for_checkout != null ? item.available_for_checkout : '0';
      availableCell.classList.add('grid-item');
      row.appendChild(availableCell);

      fragment.appendChild(row);
    }
    inventoryTableBody.replaceChildren(fragment);
  } catch (err) {
    if (err.name !== 'AbortError') console.error("fetchInventoryOverview error:", err);
  }
}

async function fetchJobQueueOverview(_signal) { return null; }

async function postNote() {
  const noteTextArea = document.getElementById('note-text');
  if (!noteTextArea) {
    alert('Note text area not found in DOM');
    return;
  }
  const noteContent = noteTextArea.innerHTML.trim();
  const noteData = {
    note_type: 'general',
    note: noteContent
  };
  try {
    const response = await fetch('/api/notes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(noteData),
      credentials: 'same-origin'
    });
    if (!response.ok) {
      throw new Error(`Failed to post note: ${response.statusText}`);
    }
    await fetchNotes();
  } catch (err) {
    console.error("postNote error:", err);
  }
}