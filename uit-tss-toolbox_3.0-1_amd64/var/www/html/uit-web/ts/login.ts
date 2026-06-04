let loginSubmitInProgress: boolean = false;

const loginForm = document.querySelector("#login-form") as HTMLFormElement;
const usernameInput = document.getElementById("username") as HTMLInputElement;
const passwordInput = document.getElementById("password") as HTMLInputElement;
const loginButton = document.getElementById("login-button") as HTMLButtonElement;
const usernameStar = document.getElementById("username-star") as HTMLElement;
const passwordStar = document.getElementById("password-star") as HTMLElement;
const errorMsg = document.getElementById("login-error") as HTMLElement;

function checkUsernameValidity(): void {
	const usernameValid = usernameInput.checkValidity();
	if (!usernameValid) {
		usernameStar.style.display = "inline-block";
		usernameStar.style.color = "red";
	} else {
		usernameStar.style.display = "none";
		usernameStar.style.color = "black";
	}
}

function checkPasswordValidity(): void {
	const passwordValid = passwordInput.checkValidity();
	if (!passwordValid) {
		passwordStar.style.display = "inline-block";
		passwordStar.style.color = "red";
	} else {
		passwordStar.style.display = "none";
		passwordStar.style.color = "black";
	}
}

checkUsernameValidity();
usernameInput.addEventListener("keyup", () => {
	checkUsernameValidity();
});

checkPasswordValidity();
passwordInput.addEventListener("keyup", () => {
	checkPasswordValidity();
});

loginForm.addEventListener("submit", async (event) => {
	if (loginSubmitInProgress) return;
	loginSubmitInProgress = true;

	event.preventDefault();
	const usernameValid = usernameInput.reportValidity();
	const passwordValid = passwordInput.reportValidity();
	const formData = new FormData(loginForm);
	if ((!formData.has("username") && formData.get("username") === null) || (!formData.has("password") && formData.get("password") === null)) {
		console.log("Username or password not provided");
		loginSubmitInProgress = false;
		return;
	}

	const providedUsername: string = formData.get("username") as string;
	const providedPassword: string = formData.get("password") as string;
	if (providedUsername === "" || providedPassword === "") {
		console.log("Username or password is empty");
		loginSubmitInProgress = false;
		return;
	}
	if (providedUsername.length > 20 || providedPassword.length > 64) {
		console.log("Username or password exceeds maximum length");
		loginSubmitInProgress = false;
		return;
	}
	if (providedUsername.length < 3 || providedPassword.length < 8) {
		console.log("Username or password does not meet minimum length requirements");
		loginSubmitInProgress = false;
		return;
	}
	if (!usernameValid || !passwordValid) {
		console.log("Invalid formatting in username or password\nUsername: " + usernameValid + "\nPassword: " + passwordValid);
		loginSubmitInProgress = false;
		return;
	}

	try {
		const jsonObj = {
			username: await generateSHA256Hash(providedUsername),
			password: await generateSHA256Hash(providedPassword)
		};
		if (!jsonObj || jsonObj.username.length !== 64 || jsonObj.password.length !== 64) throw new Error('Invalid hash format for username and/or password');

		const base64Payload = jsonToBase64(JSON.stringify(jsonObj));
		if (!base64Payload || base64Payload.length === 0) throw new Error('Failed to encode login payload json to base64');

		const response = await fetch('/login', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json',
			'Content-Transfer-Encoding': 'base64'
			},
			credentials: "same-origin",
			body: base64Payload
		});

		if (!response.ok) {
			if (response.status === 400 || response.status === 401 || response.status === 403) {
				errorMsg.style.display = "block";
				errorMsg.innerText = "Invalid username or password.";
			} else {
				errorMsg.style.display = "block";
				errorMsg.innerText = "An unexpected error occurred. Please try again later.";
			}
		}
		const jsonResponse: AuthStatusResponse = await response.json();
		if (!jsonResponse || typeof jsonResponse !== "object") throw new Error("Error parsing server response JSON")
		if (jsonResponse.status && jsonResponse.status.toLowerCase() !== "authenticated") {
			errorMsg.style.display = "block";
			errorMsg.innerText = "Authentication failed. Please check your credentials and try again.";
			throw new Error("Authentication failed: " + (jsonResponse.status ?? "unknown error"));
		}
		if (jsonResponse.expires_at === null || jsonResponse.expires_at <= new Date()) {
			errorMsg.style.display = "block";
			errorMsg.innerText = "Invalid response from server. Please try again later.";
			throw new Error("Invalid authentication response: token is already expired or expires_at field is missing/null");
		}

		if (jsonResponse.ttl === null || jsonResponse.ttl <= 0) {
			errorMsg.style.display = "block";
			errorMsg.innerText = "Invalid response from server. Please try again later.";
			throw new Error("Invalid authentication response: ttl is missing or invalid");
		}

		const redirectQuery = new URLSearchParams(window.location.search).get("redirect") ?? "";
		const redirectURL = new URL(redirectQuery, window.location.origin);
		// if (redirectURL.pathname === "/" || redirectURL.pathname === "/logout" || !redirectURL.pathname.startsWith("/") || redirectURL.pathname.startsWith("//") || redirectURL.pathname.includes("/login")) {
		// 	window.location.href = "/dashboard";
		// 	return;
		// }
		window.location.href = window.location.origin + redirectURL.pathname + redirectURL.search;
	} catch (error) {
		console.error('There was a problem with the fetch operation:', error);
		return;
	} finally {
		loginSubmitInProgress = false;
	}
});