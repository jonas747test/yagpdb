$(function(){
	if (visibleURL) {
		console.log("Should navigate to", visibleURL);
		window.history.replaceState("", "", visibleURL);
	}

	addListeners(false);

	window.onpopstate = function (evt, a) {
       	$("#main-content").html('<div class="loader">Loading...</div>');
		navigate(window.location.pathname, "GET", null, false)
        // Handle the back (or forward) buttons here
        // Will NOT handle refresh, use onbeforeunload for this.
    };
})

var currentlyLoading = false;
function navigate(url, method, data, updateHistory){
	    if (currentlyLoading) {return;}
	    currentlyLoading = true;
	    var evt = new CustomEvent('customnavigate', { url: url });
		window.dispatchEvent(evt);

		if (url[0] !== "/") {
			url = window.location.pathname + url;
		}

		console.log("Navigating to "+url);
		var shownURL = url;
		// Add the partial param
		var index = url.indexOf("?")
		if (index !== -1) {
			url += "&partial=1"	
		}else{
			url += "?partial=1"	
		}

		var req = new XMLHttpRequest();
        req.addEventListener("load", function(){
        	currentlyLoading = false;
			if (this.status != 200) {
            	window.location.href = '/';
            	return;
			}

			$("#main-content").html(this.responseText);
			if (updateHistory) {	
				window.history.pushState("", "", shownURL);
			}
			updateSelectedMenuItem();
			addListeners(true);
			if (typeof ga !== 'undefined') {
				ga('send', 'pageview', window.location.pathname);
				console.log("Sent pageview")
			}
        });

        req.addEventListener("error", function(){
            window.location.href = '/';
            currentlyLoading = false;
        });

        req.open(method, url);
        
        if (data) {
            req.setRequestHeader("content-type", "application/x-www-form-urlencoded");
            req.send(data);
        }else{
            req.send();
        }
	}


function addAlert(kind, msg){
	$("<div/>").addClass("row").append(
		$("<div/>").addClass("col-lg-12").append(
			$("<div/>").addClass("alert alert-"+kind).text(msg)
		)
	).appendTo("#alerts");
}

function clearAlerts(){
	$("#alerts").empty();
}
function addListeners(partial){
	var selectorPrefix = "";
	if (partial) {
		selectorPrefix = "#main-content ";
	}

	////////////////////////////////////////
	// Async partial page loading handling
	///////////////////////////////////////
	$( selectorPrefix + ".nav-link" ).click(function( event ) {
		console.log("Clicked the link");
		event.preventDefault();
		
	    if (currentlyLoading) {return;}
			
		var link = $(this);
		
		var url = link.attr("href");
		navigate(url, "GET", null, true);

		$("#main-content").html('<div class="loader">Loading...</div>');
	});

	var forms = $(selectorPrefix + "form");
	
	forms.each(function(i, elem){
		elem.onsubmit = submitform;
	})

	$(selectorPrefix + ".btn-danger").click(function(evt){
		var target = $(evt.target);
		if(target.attr("noconfirm") !== undefined){
			return;
		}

		if ($(evt.target).attr("noconfirm")) {
			console.log("no confirm")
			return
		}

		// console.log("aaaaa", evt, evt.preventDefault);
		if(!confirm("Are you sure you want to do this?")){
			evt.preventDefault(true);
			evt.stopPropagation();
		}
		// alert("aaa")
	})

	$(selectorPrefix + '[data-toggle="popover"]').popover()
	$(selectorPrefix + '[data-toggle="popover"]').click(function(evt){
		$('[data-toggle="popover"]').each(function(i, elem){
			// console.log(elem, elem == evt.target);
			if (evt.currentTarget == elem) {
				return;
			}
			$(elem).popover('hide');
		})
	})

	function submitform(evt){

		var possibilities = [
			"Loading!",
			"Loading...",
			"Loading :D",
			"Not Loading!",
			"Unloading!",
			"Burning down your house!",
			"Having a bbq!",
			"Taking it slow...",
			"Snack break!",
			"How was your day?",
			"HAHAHAHAHA Are you fast enough to read his? you're an idiot.",
			"Yo listen up here's a story, About a little guy that lives in a blue world, And all day and all night and everything he sees, Is just blue like him inside and outside, Blue his house with a blue little window, And a blue corvette, And everything is blue for him and himself, And everybody around, 'Cause he ain't got nobody to listen to",
			"Am loading sir o7",
			"Wonder what this button does?",
			"Hmmm this sure is taking a long time",
			"Wanna go out on a date?",
			"Am i loading?",
			"Click me harder boi ;)	)))",
		]

		for(var i = 0; i < evt.target.elements.length; i++){
			var control = evt.target.elements[i];
			if (control.type === "submit") {
				$(control).addClass("disabled").attr("disabled");
				var endless = getRandomInt(0, possibilities.length-1)
				$(control).text(possibilities[endless]);
			}						
		}
	}

	function getRandomInt(min, max) {
		min = Math.ceil(min);
		max = Math.floor(max);
		return Math.floor(Math.random() * (max - min)) + min;
	}

}
