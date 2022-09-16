document.querySelectorAll(".exerciseItem").forEach((item) => {
  item.addEventListener("click", (e) => {
    const challenge = Number.parseInt(e.target.dataset.challengeId);

    const currentlyChecked = e.target.classList.contains("clicked");

    fetch("/challenge/", {
      method: "POST",
      body: JSON.stringify({ challenge, checked: !currentlyChecked }),
    }).then((response) => {
      response.json().then(() => {
        item.classList.toggle("clicked");
      });
    });
  });
});
