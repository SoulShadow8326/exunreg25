function Signal(key, initialValue) {
  let value = initialValue;
  let onChange = null;
  return {
    Value: function () { return value; },
    setValue: function (newValue) { value = newValue; if (onChange) onChange(); },
    set onChange(callback) { onChange = callback; }
  };
}

var chatSignal = Signal('chatOpenState', 'close');

function setupChatSignalHandlers() {
  const chatToggleBtn = document.getElementById("chatToggleBtn");
  const chatPopup = document.getElementById("chatPopup");
  const chatCloseBtn = document.getElementById("chatCloseBtn");
  const chatMinimizeBtn = document.getElementById("chatMinimizeBtn");
  let overlay = document.querySelector('.chat-overlay');
  if (!overlay) {
    overlay = document.createElement('div');
    overlay.className = 'chat-overlay';
    document.body.appendChild(overlay);
  }
  if (!chatToggleBtn || !chatPopup) {
    console.error('Chat elements not found:', { chatToggleBtn: !!chatToggleBtn, chatPopup: !!chatPopup });
    return;
  }
  chatSignal.onChange = function () {
    if (chatSignal.Value() === "open") {
      chatPopup.style.display = "flex";
      if (window.matchMedia('(max-width: 768px)').matches) {
        overlay.style.display = 'block';
        document.body.style.overflow = 'hidden';
      }
      chatToggleBtn.style.opacity = 0;
      chatToggleBtn.style.transform = "scale(0)";
      setTimeout(function () {
        chatPopup.style.opacity = 1;
        if (window.matchMedia('(max-width: 768px)').matches) {
          chatPopup.style.transform = "translate(-50%, -50%) scale(1)";
        } else {
          chatPopup.style.transform = "translateY(0px)";
        }
        chatToggleBtn.style.display = "none";
      }, 10);
      if (typeof refreshChatContent === 'function') refreshChatContent();
      const messagecontainer = document.getElementById("messagecontainer");
      setTimeout(function () {
        if (messagecontainer) messagecontainer.scrollTop = messagecontainer.scrollHeight;
      }, 200);
    } else {
      if (overlay) {
        overlay.style.display = 'none';
        document.body.style.overflow = '';
      }
      chatToggleBtn.style.display = "block";
      chatPopup.style.opacity = 0;
      if (window.matchMedia('(max-width: 768px)').matches) {
        chatPopup.style.transform = "translate(-50%, -50%) scale(0.95)";
      } else {
        chatPopup.style.transform = "translateY(900px)";
      }
      setTimeout(function () {
        chatPopup.style.display = "none";
        chatToggleBtn.style.opacity = 1;
        chatToggleBtn.style.transform = "scale(1)";
      }, 400);
    }
  };
  if (chatToggleBtn) {
    chatToggleBtn.addEventListener("click", function () {
      chatSignal.setValue("open");
    });
  }
  if (chatCloseBtn) {
    chatCloseBtn.addEventListener("click", function () {
      chatSignal.setValue("close");
    });
  }
  if (chatMinimizeBtn) {
    chatMinimizeBtn.addEventListener("click", function () {
      chatSignal.setValue("close");
    });
  }

  overlay.addEventListener('click', function (e) {
    if (window.matchMedia('(max-width: 768px)').matches) {
      chatSignal.setValue('close');
    }
  });
}

document.addEventListener('DOMContentLoaded', function () {
  setupChatSignalHandlers();
  (async function loadFAQ(){
    try{
      const resp = await fetch('/data/faq.json');
      if(!resp.ok) return;
      const j = await resp.json();
      const container = document.getElementById('faqContainer');
      if(!container || !j || !j.faq) return;
      const wrapper = document.createElement('div');
      wrapper.className = 'faq-wrapper';
      j.faq.forEach(item => {
        const q = document.createElement('div');
        q.className = 'faq-item';
        const qh = document.createElement('h3');
        qh.className = 'faq-question';
        qh.textContent = item.question || '';
        const qa = document.createElement('p');
        qa.className = 'faq-answer';
        qa.textContent = (item.answer || '').replace(/\*\*/g,'');
        q.appendChild(qh);
        q.appendChild(qa);
        wrapper.appendChild(q);
      });
      container.appendChild(wrapper);
    }catch(e){
    }
  })();

  const form = document.getElementById('query-form');
  const input = document.getElementById('query-input');
  const results = document.getElementById('results');
  const clearBtn = document.getElementById('clear-btn');
  const headerSearchInput = document.querySelector('.query-search-input');
  const headerSearchBtn = document.getElementById('querySearchBtn');

  const examples = Array.from(document.querySelectorAll('.examples .filter-btn'));
  examples.forEach(btn => btn.addEventListener('click', () => {
    input.value = btn.dataset.example || '';
    input.focus();
  }));

  if (form) {
    form.addEventListener('submit', function (e) {
      e.preventDefault();
      const q = input.value.trim();
      if (!q) return;
      sendQuery(q);
      input.value = '';
    });
  }

  if (headerSearchInput) {
    headerSearchInput.addEventListener('keydown', function(e){
      if (e.key === 'Enter'){
        const q = headerSearchInput.value.trim();
        if (!q) return;
        chatSignal.setValue('open');
        sendQuery(q);
        headerSearchInput.value = '';
      }
    });
  }
  if (headerSearchBtn) {
    headerSearchBtn.addEventListener('click', function(){
      const q = headerSearchInput ? headerSearchInput.value.trim() : '';
      if (!q) return;
      chatSignal.setValue('open');
      sendQuery(q);
      headerSearchInput.value = '';
    });
  }

  if (clearBtn) {
    clearBtn.addEventListener('click', () => {
      if (results) results.innerHTML = '<p class="text-muted">No answers yet. Ask a question to get started.</p>';
    });
  }

  const chatInputEl = document.getElementById('queryInput');
  const chatSendBtn = document.getElementById('querySendButton');
  if (chatInputEl && chatSendBtn) {
    chatSendBtn.disabled = true;
    chatInputEl.addEventListener('input', () => {
      chatSendBtn.disabled = !chatInputEl.value.trim();
    });
    chatInputEl.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        const q = chatInputEl.value.trim();
        if (!q) return;
        sendQuery(q);
        chatInputEl.value = '';
        chatSendBtn.disabled = true;
      }
    });
    chatSendBtn.addEventListener('click', () => {
      const q = chatInputEl.value.trim();
      if (!q) return;
      sendQuery(q);
      chatInputEl.value = '';
      chatSendBtn.disabled = true;
    });
  }

  function addQueryToResults(query) {
    if (!results) return;
    const container = document.createElement('div');
    container.className = 'answer-card';

    const qElem = document.createElement('div');
    qElem.className = 'answer-query';
    qElem.textContent = query;

    const aElem = document.createElement('div');
    aElem.className = 'answer-body';
    aElem.textContent = generateMockAnswer(query);

    container.appendChild(qElem);
    container.appendChild(aElem);

    results.prepend(container);
  }

  function renderChatMessage(text, from) {
    const container = document.getElementById('querysContainer') || document.getElementById('messagecontainer');
    if (!container) return;
    const msg = document.createElement('div');
    msg.className = 'chat-message';
    if (from === 'user') msg.classList.add('user');
    else msg.classList.add('admin');
    const content = document.createElement('div');
    content.className = 'chat-message-content';
    content.textContent = text;
    msg.appendChild(content);
    container.appendChild(msg);
    container.scrollTop = container.scrollHeight;
  }

  function stripMarkdown(md) {
    if (!md) return md;
    let s = md;
    s = s.replace(/```[\s\S]*?```/g, '');
    s = s.replace(/`([^`]*)`/g, '$1');
    s = s.replace(/!\[([^\]]*)\]\([^\)]*\)/g, '$1');
    s = s.replace(/\[([^\]]+)\]\([^\)]+\)/g, '$1');
    s = s.replace(/(^|\n)#{1,6}\s*(.*)/g, '$1$2');
    s = s.replace(/(\*\*|__)(.*?)\1/g, '$2');
    s = s.replace(/(\*|_)(.*?)\1/g, '$2');
    s = s.replace(/(^|\n)>\s?/g, '$1');
    s = s.replace(/(^|\n)[\-\*\+]\s+/g, '$1');
    s = s.replace(/(^|\n)\d+\.\s+/g, '$1');
    s = s.replace(/\r\n/g, '\n');
    s = s.replace(/\n{2,}/g, '\n\n');
    s = s.trim();
    return s;
  }

  async function sendQuery(q) {
    const chatContainer = document.getElementById('querysContainer') || document.getElementById('messagecontainer');
    if (chatContainer) {
      const empties = chatContainer.querySelectorAll('.empty-state');
      empties.forEach(e => e.style.display = 'none');
    }
    renderChatMessage(q, 'user');
    renderChatMessage('Thinking...', 'bot');
    try {
      const res = await fetch('/api/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: q })
      }); 
      const data = await res.json();

      const container = document.getElementById('querysContainer') || document.getElementById('messagecontainer');
      if (container) {
        const nodes = Array.from(container.querySelectorAll('.chat-message'));
        for (let i = nodes.length - 1; i >= 0; i--) {
          if (nodes[i].textContent && nodes[i].textContent.includes('Thinking...')) {
            container.removeChild(nodes[i]);
            break;
          }
        }
      }
      if (data && data.answer) {
        renderChatMessage(stripMarkdown(data.answer), 'bot');
      } else {
        renderChatMessage('No answer returned.', 'bot');
      }
    } catch (err) {
      renderChatMessage('Error contacting server.', 'bot');
    }
  }

  function generateMockAnswer(q) {
    const text = q.toLowerCase();
    if (text.includes('schedule') || text.includes('when')) {
      return 'Final schedule is in the Schedule document. Please check the Schedule link in the navbar.';
    }
    if (text.includes('register') || text.includes('registration')) {
      return 'Registration is open on the website. Create an account and fill the registration form. Team events require team query to register and add members.';
    }
    if (text.includes('eligib') || text.includes('eligibility')) {
      return 'Eligibility varies by event. Most events are open to high-school students; check the specific event page for details.';
    }
    if (text.includes('rule') || text.includes('rulebook')) {
      return 'The rulebook is available in the Brochure. For clarifications, email exun@dpsrkp.net.';
    }
    return 'No exact match found in local knowledge. Try rephrasing, or check the Brochure and Schedule links in the navbar.';
  }
});
