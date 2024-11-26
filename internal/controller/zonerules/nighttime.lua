function Evaluate(_, zone, _)
		if zone == "auto" then
			return "auto", 0, "no manual setting detected"
		end
		local nightMode = IsInRange(0, 0, 6, 0)
		local delay  = 0
		if not nightMode == true then
			delay = SecondsTill(0, 0)
		end
		return "auto", delay, "manual setting detected"
end
