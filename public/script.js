document.getElementById('shorten-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const url = document.getElementById('url-input').value;
    const res = await fetch('/api/shorten', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url })
    });
    if (res.ok) {
        const data = await res.json();
        document.getElementById('short-url').href = data.short;
        document.getElementById('short-url').textContent = data.short;
        document.getElementById('result').classList.remove('hidden');
    } else {
        alert('Error shortening URL');
    }
});

document.getElementById('copy-btn').addEventListener('click', () => {
    const shortUrl = document.getElementById('short-url').textContent;
    navigator.clipboard.writeText(shortUrl);
    alert('Copied!');
});

document.getElementById('stats-btn').addEventListener('click', async () => {
    const code = document.getElementById('stats-code').value.trim();
    if (!code) return;
    const res = await fetch('/api/stats?code=' + encodeURIComponent(code));
    const div = document.getElementById('stats-result');
    if (res.ok) {
        const data = await res.json();
        div.innerHTML = `<p><strong>Long URL:</strong> <a href="${data.long}" target="_blank">${data.long}</a></p>
                         <p><strong>Clicks:</strong> ${data.clicks}</p>`;
    } else {
        div.innerHTML = '<p>Not found.</p>';
    }
});
