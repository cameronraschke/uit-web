let isLoggingOut = false;

const authChannel = new BroadcastChannel('auth');
authChannel.onmessage = function(event) {
  console.log(event);
  if (event.data.cmd === 'logout') {
    logout();
  }
};

const logoutButton = document.getElementById("logout-button");
logoutButton.addEventListener("click", function(event) {
  event.preventDefault();
  logout();
});

async function logout() {
  if (isLoggingOut) return;
  isLoggingOut = true;
  logoutButton.disabled = true;
  authChannel.postMessage({cmd: 'logout'});
  localStorage.clear();
  sessionStorage.clear();
  try {
    const response = await fetch("/logout", {
      method: "GET",
      credentials: "same-origin"
    });
    if (!response.ok) {
      alert("Logout failed, try again.");
    }
  } catch (error) {
    console.error("Logout error:", error);
  } finally {
    isLoggingOut = false;
    logoutButton.disabled = false;
    window.location.replace("/login.html");
  }
}