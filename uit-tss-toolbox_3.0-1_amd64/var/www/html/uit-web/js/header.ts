const currentPath = window.location.pathname;

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
	const tagLookup = document.querySelector("#global-client-lookup") as HTMLInputElement;
	const globalSearchDatalist = document.querySelector("#global-client-lookup-datalist") as HTMLDataListElement;
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
			renderTagOptions(globalSearchDatalist, window.allTags, 10);
			return;
		}
		const filteredTags = window.allTags.filter(tag => tag.toString().includes(inputVal));
		renderTagOptions(globalSearchDatalist, filteredTags, 10);
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