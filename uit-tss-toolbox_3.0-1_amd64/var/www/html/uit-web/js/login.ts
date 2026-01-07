let loginSubmitInProgress: boolean = false;

interface LoginInfo {
	username: string;
	password: string;
}

const loginForm = document.querySelector("#login-form") as HTMLFormElement;
const usernameInput = document.getElementById("username") as HTMLInputElement;
const passwordInput = document.getElementById("password") as HTMLInputElement;
const loginButton = document.getElementById("login-button") as HTMLButtonElement;
const usernameStar = document.getElementById("username-star") as HTMLElement;
const passwordStar = document.getElementById("password-star") as HTMLElement;

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
    if (!formData.has("username") || !formData.has("password")) {
        console.log("Username or password not provided");
        loginSubmitInProgress = false;
        return;
    }

		const username: string = (formData.get("username") as string).trim();
		const password: string = (formData.get("password") as string).trim();
    if (username === "" || password === "") {
        console.log("Username or password is empty");
        loginSubmitInProgress = false;
        return;
    }
    if (username.length > 20 || password.length > 64) {
        console.log("Username or password is too long");
        loginSubmitInProgress = false;
        return;
    }
    if (username.length < 3 || password.length < 8) {
        console.log("Username or password is too short");
				loginSubmitInProgress = false;
        return;
    }
    if (/\s/.test(username) || /\s/.test(password)) {
        console.log("Username or password contains whitespace");
				loginSubmitInProgress = false;
        return;
    }
    if (!usernameValid || !passwordValid) {
        console.log("Invalid formatting in username or password\nUsername: " + usernameValid + "\nPassword: " + passwordValid);
				loginSubmitInProgress = false;
        return;
    }

    try {
        const jsonArr: LoginInfo = {
            username: await generateSHA256Hash(username),
            password: await generateSHA256Hash(password)
        };


        const jsonData = JSON.stringify(jsonArr);
        if (!jsonData || jsonData.length === 0) throw new Error('No data to send to login API');

        const base64Payload = jsonToBase64(jsonData);
        if (!base64Payload || base64Payload.length === 0) throw new Error('Failed to encode login payload to base64');

        const response = await fetch('/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json',
              'Content-Transfer-Encoding': 'base64'
            },
            credentials: "include",
            body: base64Payload
        });

        if (!response.ok) {
          const errorMsg = document.getElementById("login-error");
            if (errorMsg) {
              if (response.status === 401 || response.status === 403 || response.status === 400) {
                errorMsg.style.display = "block";
                errorMsg.innerText = "Invalid username or password.";
              } else {
                errorMsg.style.display = "block";
                errorMsg.innerText = "An unexpected error occurred. Please try again later.";
              }
            } else {
              throw new Error('Network response was not ok: ' + response.statusText);
            }
        }
        const data = await response.json();
        if (!data || (typeof data === "object" && Object.keys(data).length === 0) || !data.token || data.token.length === 0) {
            throw new Error('No data returned from login API');
        }

        await setKeyFromIndexDB("bearerToken", data.token);

        window.location.href = "/dashboard";
    } catch (error) {
        console.error('There was a problem with the fetch operation:', error);
        loginSubmitInProgress = false;
    } finally {
        loginSubmitInProgress = false;
    }
});

async function setKeyFromIndexDB(key: string, value: string): Promise<void> {
  if (typeof key !== "string" || key.trim() === "" ||
      typeof value !== "string" || value.trim() === "") {
    throw new Error("Invalid key/value");
  }

  const db = await openTokenDB() as IDBDatabase;

  return new Promise((resolve, reject) => {
    try {
      const tx = db.transaction("uitTokens", "readwrite");
      const store = tx.objectStore("uitTokens");
      store.put({ tokenType: key, value: value });
      tx.oncomplete = () => resolve(void 0);
      tx.onerror = (e) => reject((e.target as IDBRequest).error);
      tx.onabort = (e) => reject((e.target as IDBRequest).error);
    } catch (err) {
      reject(err);
    }
  });
}