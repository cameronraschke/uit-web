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

async function fetchHeader(): Promise<string | null> {
  try {
    const headerContent: string | null = await fetchData("/header", true);
    return headerContent;
  } catch (error) {
    console.error("Error fetching header:", error);
    return null;
  }
}

function drawHeader(headerHTMLContent: string | null = null): void {
	const header = document.querySelector("#uit-header");
	if (header === null) {
		console.log("Header element not found, cannot render header");
		return;
	}
  try {
    header.innerHTML = headerHTMLContent ?? "";
  } catch (error) {
    console.error("Error fetching header:", error);
  }
}

function initHeader() {
	const globalLookupForm: HTMLFormElement | null = document.querySelector("#global-client-lookup-form");
	const tagLookup: HTMLInputElement | null = document.querySelector("#global-client-lookup");
	const globalSearchDatalist: HTMLDataListElement | null = document.querySelector("#global-client-lookup-datalist");
	const keyupDebounceMs = 75;
	let keyupDebounceTimer: number | undefined;
	let cachedTagNumbers: number[] = [];

	if (globalLookupForm === null) {
		console.warn("Global lookup form not found, skipping tag search initialization");
		return;
	}
	if (tagLookup === null) {
		console.warn("Tag lookup input not found, skipping tag search initialization");
		return;
	}
	if (globalSearchDatalist === null) {
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
	const logoutButton: HTMLAnchorElement | null = document.querySelector("#menu-logout-button");
	if (logoutButton === null) {
		console.warn("Logout button element not found, skipping logout initialization");
		return;
	}
	
	logoutButton.addEventListener("click", (event) => {
		event.preventDefault();
		localStorage.clear();
		sessionStorage.clear();
		headerAuthChannel.postMessage({cmd: 'logout'});

    // let logoutUrl = "/logout";
    // if (currentPath !== "/login" && currentPath !== "/logout") {
		// 	logoutUrl += "?redirect=" + encodeURIComponent(currentPath + window.location.search);
    // }
		window.location.replace("/logout");
	});
}

document.addEventListener("DOMContentLoaded", async () => {
	try {
		const headerContent = await fetchHeader();
		drawHeader(headerContent);
		initLogout();
		initHeader();
	} catch (error) {
		console.error("Error in drawHeader:", error);
	}
	for (const [relativePath, menuElementId] of pathMap) {
		const menuItem = document.getElementById(menuElementId);
		if (menuItem === null) {
			console.warn(`Menu item with ID ${menuElementId} not found, skipping active state check for this item.`);
			continue;
		}

		if (currentPath === relativePath) {
			menuItem.classList.add("active");
		} else if (menuItem.classList.contains('active')) {
			menuItem.classList.remove("active");
		}
	}
});