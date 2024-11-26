function Evaluate(_, zone, _)
	if zone == "auto" then
		return "auto", 0, "no manual setting detected"
	end
	return "auto", 3600, "manual setting detected"
end
