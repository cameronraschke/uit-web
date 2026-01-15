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

document.addEventListener("DOMContentLoaded", () => {
	drawHeader().then(() => {
		for (const [path, elementId] of pathMap) {
			const menuItem = document.getElementById(elementId);
			if (!menuItem) continue;

			console.log(`Checking path: ${path} against currentPath: ${currentPath}`);
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
	);
});