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

	tagLookup.addEventListener('keyup', () => {
		const inputVal = tagLookup.value.trim();
		if (inputVal.length === 0) {
			renderTagOptions(globalSearchDatalist, window.globalLookupResults.flatMap(entry => entry.entries || []).map(entry => entry.tagnumber).filter((tag): tag is number => typeof tag === "number"), 0);
			return;
		}
		const globalSearchValues = window.globalLookupResults.flatMap(entry => entry.entries || []).map(entry => entry.tagnumber).filter((tag): tag is number => typeof tag === "number").filter(tag => tag.toString().includes(inputVal));
		renderTagOptions(globalSearchDatalist, globalSearchValues, 10);
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