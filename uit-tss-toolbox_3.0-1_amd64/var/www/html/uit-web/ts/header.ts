const currentPath = window.location.pathname;

const headerAuthChannel = new BroadcastChannel('auth');
headerAuthChannel.onmessage = function(event) {
  if (event.data.cmd === 'logout') {
    if (window.location.pathname !== '/login' && window.location.pathname !== '/logout') {
			localStorage.clear();
			sessionStorage.clear();
			window.location.replace("/logout");
    }
  }
};

const pathMap : Map<string, string> = new Map([
	["/dashboard", "menu-dashboard"],
	["/inventory", "menu-inventory"],
	["/checkouts", "menu-checkouts"],
	["/job-queue", "menu-job-queue"],
	["/reports", "menu-reports"]
]);

async function drawHeader() {
	const header = document.getElementById("uit-header");
	if (!header) return;
  try {
    const headerContent = await fetchData("/header", true);
    header.innerHTML = headerContent;
  } catch (error) {
    console.error("Error fetching header:", error);
  }
}

function initHeader() {
	const globalLookupForm = document.querySelector("#global-client-lookup-form") as HTMLFormElement;
	const tagLookup = document.querySelector("#global-client-lookup") as HTMLInputElement;
	const globalSearchDatalist = document.querySelector("#global-client-lookup-datalist") as HTMLDataListElement;
	const keyupDebounceMs = 75;
	let keyupDebounceTimer: number | undefined;
	let cachedTagNumbers: number[] = [];

	if (!globalLookupForm) {
		console.warn("Global lookup form not found, skipping tag search initialization");
		return;
	}
	if (!tagLookup) {
		console.warn("Tag lookup input not found, skipping tag search initialization");
		return;
	}
	if (!globalSearchDatalist) {
		console.warn("Global search datalist not found, skipping tag search initialization");
		return;
	}

	const rebuildTagCache = (entriesSource?: any[]) => {
		const source = Array.isArray(entriesSource) ? entriesSource : window.globalLookupResults;
		cachedTagNumbers = Array.isArray(source)
			? source
				.flatMap((entry: any) => entry.entries || [])
				.map((entry: any) => entry.tagnumber)
				.filter((tag: any): tag is number => typeof tag === "number")
			: [];
	};

	const renderLookupOptions = () => {
		if (!cachedTagNumbers.length) {
			rebuildTagCache();
		}
		const inputVal = tagLookup.value.trim();
		if (inputVal.length === 0) {
			renderTagOptions(globalSearchDatalist, cachedTagNumbers, 0);
			return;
		}
		const globalSearchValues = cachedTagNumbers.filter(tag => tag.toString().includes(inputVal));
		renderTagOptions(globalSearchDatalist, globalSearchValues, 10);
	};

	rebuildTagCache();
	tagLookup.addEventListener('keyup', () => {
		if (keyupDebounceTimer !== undefined) {
			clearTimeout(keyupDebounceTimer);
		}
		keyupDebounceTimer = window.setTimeout(renderLookupOptions, keyupDebounceMs);
	});

	document.addEventListener("tags:loaded", (event: Event) => {
		const customEvent = event as CustomEvent<{ entries: any[] }>;
		const rawEntries = (customEvent && customEvent.detail && Array.isArray(customEvent.detail.entries)) ? customEvent.detail.entries : window.globalLookupResults;
		rebuildTagCache(rawEntries);
		renderLookupOptions();
	});

	globalLookupForm.addEventListener('submit', (event) => {
		event.preventDefault();
		const tagValue = tagLookup.value.trim();
		if (tagValue.length === 0) {
			return;
		}
		window.location.href = `/inventory?tagnumber=${encodeURIComponent(tagValue)}&update=true`;
	});
}

function initLogout() {
	const logoutButton = document.getElementById("menu-logout-button") as HTMLAnchorElement;
	if (!logoutButton) return;
	
	logoutButton.addEventListener("click", (event) => {
		event.preventDefault();
		headerAuthChannel.postMessage({cmd: 'logout'});
		localStorage.clear();
		sessionStorage.clear();

    let logoutUrl = "/logout";
    if (currentPath !== "/login" && currentPath !== "/logout") {
        logoutUrl += "?redirect=" + encodeURIComponent(currentPath + window.location.search);
    }
    window.location.href = logoutUrl;
	});
}

document.addEventListener("DOMContentLoaded", () => {
	drawHeader().then(() => {
		for (const [path, elementId] of pathMap) {
			const menuItem = document.getElementById(elementId);
			if (!menuItem) continue;

			if (currentPath === path) {
				menuItem.classList.add("active");
			} else if (menuItem.classList.contains('active')) {
				menuItem.classList.remove("active");
			}
		}
		initLogout();
	}).catch(
		(error) => {
			console.error("Error in drawHeader:", error);
		}
	).then(() => {
		initHeader();
	}).catch(
		(error) => {
			console.error("Error in initHeader:", error);
		}
	);
});