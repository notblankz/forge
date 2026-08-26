// Adds a pinned "Copy" button to the prompt code block on the AI theme
// generation page. Loaded only when a page sets `copy_prompt: true` in its
// front matter, so it targets the first (and only) fenced block on that page.
(function () {
  var ICON_COPY =
    '<svg class="copy-ico" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="11" height="11" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>';
  var ICON_DONE =
    '<svg class="copy-ico" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>';

  function label(text) {
    return '<span class="copy-label">' + text + "</span>";
  }

  function init() {
    var pre = document.querySelector(".content pre");
    if (!pre) return;

    var text = pre.innerText; // captured before the button is added to the DOM

    // Wrap the pre so the button anchors to a non-scrolling container.
    var wrap = document.createElement("div");
    wrap.className = "code-wrap";
    pre.parentNode.insertBefore(wrap, pre);
    wrap.appendChild(pre);

    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "copy-btn";
    btn.innerHTML = ICON_COPY + label("Copy");
    btn.setAttribute("aria-label", "Copy prompt to clipboard");
    wrap.appendChild(btn);

    var reset;
    function flash(ok) {
      btn.innerHTML = (ok ? ICON_DONE : ICON_COPY) + label(ok ? "Copied" : "Press Ctrl+C");
      btn.classList.toggle("copied", ok);
      clearTimeout(reset);
      reset = setTimeout(function () {
        btn.innerHTML = ICON_COPY + label("Copy");
        btn.classList.remove("copied");
      }, 1600);
    }

    function legacyCopy() {
      var ta = document.createElement("textarea");
      ta.value = text;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.appendChild(ta);
      ta.select();
      var ok = false;
      try { ok = document.execCommand("copy"); } catch (e) {}
      document.body.removeChild(ta);
      return ok;
    }

    btn.addEventListener("click", function () {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(
          function () { flash(true); },
          function () { flash(legacyCopy()); }
        );
      } else {
        flash(legacyCopy());
      }
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
