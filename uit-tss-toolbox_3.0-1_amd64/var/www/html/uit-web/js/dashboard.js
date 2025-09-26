let dashboardPollController = null;

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
});

async function fetchDashboardData(signal) {
  const tasks = [
    fetchNotes(signal), 
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
    const noteTextArea = document.getElementById('note-textarea');
    if (!noteTextArea) throw new Error('Note text area not found in DOM');
    noteTextArea.value = jsonParsed.note || '';
    return jsonParsed;
  } catch (err) {
    if (err.name !== 'AbortError') console.error("fetchNotes error:", err);
    return null;
  }
}

async function fetchInventoryOverview(_signal) { return null; }
async function fetchJobQueueOverview(_signal) { return null; }