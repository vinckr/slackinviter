const url = "https://auth.slackinviter.vinckr.com/sessions/whoami";

try {
  fetch(url, {
    method: "GET",
    credentials: "include",
  })
    .then((response) => response.json())
    .then((data) => console.log(data));
  fetch("/sessiondata", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(data),
  });
} catch (error) {
  console.error(error.message); // Error message
}
