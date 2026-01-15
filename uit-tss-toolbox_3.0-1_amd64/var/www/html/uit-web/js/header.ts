const menuDashboard = document.getElementById("menu-dashboard") as HTMLAnchorElement;
const menuInventory = document.getElementById("menu-inventory") as HTMLAnchorElement;
const menuCheckouts = document.getElementById("menu-checkouts") as HTMLAnchorElement;
const menuJobQueue = document.getElementById("menu-job-queue") as HTMLAnchorElement;
const menuReports = document.getElementById("menu-reports") as HTMLAnchorElement;
const menuTagLookup = document.getElementById("menu-tag-lookup") as HTMLAnchorElement;
const menuLogout = document.getElementById("menu-logout-button") as HTMLAnchorElement;
const currentPath = window.location.pathname;

const pathMap : { [key: string]: HTMLAnchorElement } = {
	"/dashboard": menuDashboard,
	"/inventory": menuInventory,
	"/checkouts": menuCheckouts,
	"/job-queue": menuJobQueue,
	"/reports": menuReports
};

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
drawHeader();

document.addEventListener("DOMContentLoaded", () => {
	for (const [path, menuItem] of Object.entries(pathMap)) {
		if (currentPath === path) {
			menuItem.classList.add("active");
		} else {
			menuItem.classList.remove("active");
		}
	}
});