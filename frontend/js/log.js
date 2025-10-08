document.addEventListener('DOMContentLoaded', function(){
  var input = document.getElementById('logFilter')
  if(!input) return
  input.addEventListener('input', function(){
    var q = input.value.toLowerCase().trim()
    var rows = document.querySelectorAll('.log-table tbody tr')
    rows.forEach(function(r){
      var text = r.textContent.toLowerCase()
      r.style.display = q === '' || text.indexOf(q) !== -1 ? '' : 'none'
    })
  })
})
