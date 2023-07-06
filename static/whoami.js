async function getSession() {
  const url = "https://auth.slackinviter.vinckr.com/sessions/whoami"; // Replace with the actual URL

  try {
    const response = await fetch(url, { credentials: "include" });
    if (response.ok) {
      const data = await response.json();
      console.log(data); // Session object
    } else {
      throw new Error(`Request failed with status ${response.status}`);
    }
  } catch (error) {
    console.error(error.message); // Error message
  }
}

getSession();
