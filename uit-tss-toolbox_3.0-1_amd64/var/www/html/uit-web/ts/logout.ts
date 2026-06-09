var isLoggingOut: boolean = false;

const logoutAuthChannel = new BroadcastChannel('auth');
logoutAuthChannel.onmessage = function(event) {
	console.log(event);
	if (event.data.cmd === 'logout') {
		logout();
	}
};

const logoutButton = document.getElementById("menu-logout-button") as HTMLButtonElement | null;
if (logoutButton !== null) {
	logoutButton.addEventListener("click", function(event) {
		event.preventDefault();
		this.blur();
		this.disabled = true;
		logout();
	});
}

async function logout() {
	if (isLoggingOut) {
		console.warn("Logout already in progress, ignoring additional logout request.");
		return;
	}
	isLoggingOut = true;

	logoutAuthChannel.postMessage({cmd: 'logout'});
	localStorage.clear();
	sessionStorage.clear();

	let logoutUrl = "/logout";
	// const currentPath = window.location.pathname;
	// if (currentPath !== "/login" && currentPath !== "/logout") {
	// 	logoutUrl += "?redirect=" + encodeURIComponent(currentPath + window.location.search);
	// }

	window.location.replace(logoutUrl);
}

if (window.location.pathname === '/logout') {
	logout();
}