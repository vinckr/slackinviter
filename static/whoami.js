console.log("whoami.js");

async function getSession() {
  const url = "https://auth.slackinviter.vinckr.com/sessions/whoami"; // Replace with the actual URL

  try {
    const response = await axios.get(url, { withCredentials: true });
    console.log(response.data); // Session object
  } catch (error) {
    console.error(error.response.status, error.response.data); // Error response
  }
}

getSession();
