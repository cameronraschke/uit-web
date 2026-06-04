const noteSubmitButton = document.getElementById('update-note-button') as HTMLButtonElement;
const noteTextAreaEl = document.getElementById('note-text') as HTMLDivElement;
const noteTimestampEl = document.getElementById('note-last-updated') as HTMLElement;

let updatingNote = false;

document.addEventListener("DOMContentLoaded", () => {
	fetchDashboardData();
  fetchNotes();
  noteSubmitButton.addEventListener('click', async () => {
    noteTextAreaEl.contentEditable = "false";
    noteSubmitButton.disabled = true;
    if (updatingNote) return;
		updatingNote = true;
    try {
      await postNote();
		} catch (err) {
			console.error("Error updating note:", err);				
    } finally {
      updatingNote = false;
      noteTextAreaEl.contentEditable = "true";
      noteSubmitButton.disabled = false;
    }
  });
});

async function fetchDashboardData() {
	const tasks = [
		fetchInventoryOverview(), 
		fetchJobQueueOverview(),
	];
	const results = await Promise.allSettled(tasks);

	for (const r of results) {
		if (r.status === 'rejected') console.error('Task failed:', r.reason);
	}
}

async function fetchNotes() {
	if (!noteTimestampEl) {
		console.error('Note date element not found in DOM');
		return null;
	}
	if (!noteTextAreaEl) {
		console.error('Note text area not found in DOM');
		return null;
	}

  try {
    const response = await fetchData('/api/overview/note?note_type=general', true);
    if (!response || response.length === 0) throw new Error('No data received from /api/overview/note');
    const jsonParsed = JSON.parse(response);
    if (!jsonParsed || Object.keys(jsonParsed).length === 0 || (jsonParsed && typeof jsonParsed === 'object' && Object.prototype.hasOwnProperty.call(jsonParsed, '__proto__'))) {
      throw new Error('Response JSON is empty or invalid');
    }
    noteTimestampEl.innerText = jsonParsed.time ? "Note last updated: " + new Date(jsonParsed.time).toLocaleString() : '';
    noteTextAreaEl.innerHTML = jsonParsed.note || '';
    return jsonParsed;
  } catch (err) {
		alert('Error fetching notes. See console for details.');
		if (err instanceof Error) console.error("fetchNotes error:", err);
    return null;
  }
}

async function postNote() {
  if (!noteTextAreaEl) {
    alert('Note text area not found in DOM, cannot update note.');
    return;
  }
  const noteContent = noteTextAreaEl.innerHTML.trim();
  const noteData = {
    note_type: 'general',
    note: noteContent
  };
  try {
    const response = await fetch('/api/overview/note', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(noteData),
      credentials: 'same-origin',
    });
    if (!response.ok) {
      throw new Error(`Failed to post note: ${response.statusText}`);
    }
    await fetchNotes();
  } catch (err) {
    console.error("postNote error:", err);
  }
}

async function fetchInventoryOverview() {
	return;
}

async function fetchJobQueueOverview() { return null; }

