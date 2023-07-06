async function getSession() {
  const url = "https://auth.slackinviter.vinckr.com/sessions/whoami"; // Replace with the actual URL

  try {
    const response = await fetch(url, { credentials: "include" });
    if (response.ok) {
      const data = await response.json();
      console.log(data); // Session object
      await fetch("/sessiondata", {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify(data),
      });
    }
  } catch (error) {
    console.error(error.message); // Error message
  }
}
getSession();
