function Evaluate(_, zone, _)
	if zone == "auto" then
		return "auto", 0, "no manual setting detected"
	end
	return "auto", 300, "manual setting detected"
end
