function Evaluate(_, zone, _, _)
	if zone.Overlay then
    	zone.Overlay = false
	    return zone, 3600, "manual setting detected"
	end
    return zone, 0, "no manual setting detected"
end
