let isLoggingOut: boolean = false;

const logoutAuthChannel = new BroadcastChannel('auth');
logoutAuthChannel.onmessage = function(event) {
  console.log(event);
  if (event.data.cmd === 'logout') {
    logout();
  }
};

const logoutButton = document.getElementById("menu-logout-button") as HTMLButtonElement;
logoutButton.addEventListener("click", function(event) {
  event.preventDefault();
  logout();
});

async function logout() {
  if (isLoggingOut) return;
  isLoggingOut = true;
  logoutButton.disabled = true;
  logoutAuthChannel.postMessage({cmd: 'logout'});
  localStorage.clear();
  sessionStorage.clear();

  let logoutUrl = "/logout";
  const currentPath = window.location.pathname;
  if (currentPath !== "/login" && currentPath !== "/logout") {
    logoutUrl += "?redirect=" + encodeURIComponent(currentPath + window.location.search);
  }

  window.location.replace(logoutUrl);
}

if (window.location.pathname === '/logout') {
  logout();
}