function Evaluate(_, zone, _, _)
	if not zone.Manual then
		return zone, 0, "no manual setting detected"
	end
	zone.Manual = false
	return zone, 3600, "manual setting detected"
end
