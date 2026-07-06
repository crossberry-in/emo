// emo docs — sidebar active state + copy buttons
(function() {
  // Highlight active sidebar link
  var path = window.location.pathname.split('/').pop() || 'index.html';
  var links = document.querySelectorAll('.sidebar a');
  links.forEach(function(a) {
    var href = a.getAttribute('href');
    if (href === path || (path === '' && href === 'index.html')) {
      a.classList.add('active');
    }
  });

  // Copy buttons
  document.querySelectorAll('.install-cmd .copy').forEach(function(btn) {
    btn.addEventListener('click', function() {
      var cmd = btn.parentElement.querySelector('.cmd').textContent;
      navigator.clipboard.writeText(cmd).then(function() {
        var orig = btn.textContent;
        btn.textContent = '✓ Copied';
        setTimeout(function() { btn.textContent = orig; }, 1500);
      });
    });
  });

  // Syntax highlighting (minimal)
  document.querySelectorAll('pre code').forEach(function(block) {
    var html = block.innerHTML;
    // Comments
    html = html.replace(/(\/\/[^\n]*)/g, '<span style="color:#6A9955">$1</span>');
    html = html.replace(/(#[^\n]*)/g, '<span style="color:#6A9955">$1</span>');
    // Strings
    html = html.replace(/("[^"]*")/g, '<span style="color:#CE9178">$1</span>');
    // Keywords
    html = html.replace(/\b(component|state|render|style|import|from|func|var|return|if|else|for|package)\b/g,
      '<span style="color:#569CD6">$1</span>');
    block.innerHTML = html;
  });
})();
