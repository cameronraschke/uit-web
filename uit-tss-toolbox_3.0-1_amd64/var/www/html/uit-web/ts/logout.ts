var isLoggingOut: boolean = false;

const logoutAuthChannel = new BroadcastChannel('auth');
logoutAuthChannel.onmessage = function(event) {
	console.log(event);
	if (event.data.cmd === 'logout') {
		logout(undefined, false);
	}
};

const logoutButton = document.getElementById("menu-logout-button") as HTMLButtonElement | null;
if (logoutButton !== null) {
	logoutButton.addEventListener("click", function(event) {
		event.preventDefault();
		this.blur();
		this.disabled = true;
		const decodedRedirectURL = window.location.pathname + window.location.search + window.location.hash;
		logout(decodedRedirectURL);
	});
}

function logout(redirectedURL?: string, shouldBroadcast: boolean = true): void {
	if (isLoggingOut) {
		console.warn("Logout already in progress, ignoring additional logout request.");
		return;
	}
	isLoggingOut = true;

	localStorage.clear();
	sessionStorage.clear();
	if (shouldBroadcast) {
		logoutAuthChannel.postMessage({cmd: 'logout'});
	}

	const defaultRedirect = "/logout";
	const currentRelativePath = window.location.pathname;
	let fullLogoutUrl = defaultRedirect;
	if (redirectedURL && (currentRelativePath !== "/login" && currentRelativePath !== "/logout")) {
		fullLogoutUrl = defaultRedirect + "?redirect=" + encodeURIComponent(redirectedURL);
	}

	window.location.replace(fullLogoutUrl);
}

if (window.location.pathname === '/logout') {
	logout(undefined, false);
}