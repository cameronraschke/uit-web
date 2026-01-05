let dashboardPollController = null as DashboardPollController | null;
interface DashboardPollController {
	stop: () => void;
}
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
  const textArea = document.getElementById('note-text') as HTMLTextAreaElement;
  const noteSubmitButton = document.getElementById('update-note-button') as HTMLButtonElement;
  noteSubmitButton.addEventListener('click', async () => {
    textArea.disabled = true;
    noteSubmitButton.disabled = true;
    if (updatingNote) return;
    updatingNote = true;
    try {
      await postNote(new AbortController().signal);
    } finally {
      updatingNote = false;
      textArea.disabled = false;
      noteSubmitButton.disabled = false;
    }
  });
});

async function fetchDashboardData(signal: AbortSignal) {
  const tasks = [
    fetchInventoryOverview(signal), 
    fetchJobQueueOverview(signal)
  ];
  const results = await Promise.allSettled(tasks);
  for (const r of results) {
    if (r.status === 'rejected') console.error('Task failed:', r.reason);
  }
}

async function fetchNotes(signal?: AbortSignal) {
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
    if (err instanceof Error && err.name !== 'AbortError') console.error("fetchNotes error:", err);
    return null;
  }
}

async function fetchInventoryOverview(signal: AbortSignal) {
  try {
    const response = await fetchData('/api/dashboard/inventory_summary', true, { signal });
    if (!response || response.length === 0) throw new Error('No data received from /api/dashboard/inventory_summary');
    const jsonParsed = JSON.parse(response);
    if (!jsonParsed || Object.keys(jsonParsed).length === 0 || (jsonParsed && typeof jsonParsed === 'object' && Object.prototype.hasOwnProperty.call(jsonParsed, '__proto__'))) {
      throw new Error('Response JSON is empty or invalid');
    }
    const inventoryTableBody = document.getElementById('inventory-summary-body');
    if (!inventoryTableBody) throw new Error('Inventory table body element not found in DOM');

    const rows = Array.isArray(jsonParsed) ? jsonParsed : [jsonParsed];

    
    const fragment = document.createDocumentFragment();
    for (const item of rows) {
      const row = document.createElement('div');
      row.classList.add("grid-item", "row", "home");

      const modelCell = document.createElement('p');
      modelCell.textContent = item.system_model || 'N/A';
      row.appendChild(modelCell);

      const countCell = document.createElement('p');
      countCell.textContent = item.system_model_count != null ? item.system_model_count : '0';
      row.appendChild(countCell);

      const checkedOutCell = document.createElement('p');
      checkedOutCell.textContent = item.total_checked_out != null ? item.total_checked_out : '0';
      row.appendChild(checkedOutCell);

      const availableCell = document.createElement('p');
      availableCell.textContent = item.available_for_checkout != null ? item.available_for_checkout : '0';
      row.appendChild(availableCell);

      fragment.appendChild(row);
    }
    inventoryTableBody.replaceChildren(fragment);
  } catch (err) {
    if (err instanceof Error && err.name !== 'AbortError') console.error("fetchInventoryOverview error:", err);
  }
}

async function fetchJobQueueOverview(_signal: AbortSignal) { return null; }

async function postNote(signal?: AbortSignal) {
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
      credentials: 'same-origin',
      ...(signal ? { signal } : {})
    });
    if (!response.ok) {
      throw new Error(`Failed to post note: ${response.statusText}`);
    }
    await fetchNotes(signal);
  } catch (err) {
    console.error("postNote error:", err);
  }
}