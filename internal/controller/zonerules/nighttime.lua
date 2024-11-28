function Evaluate(_, zone, _, args)
		if zone == "auto" then
			return "auto", 0, "no manual setting detected"
		end
        if #args == 0 then
            args = { StartHour = 23, StartMin = 30, EndHour = 6, EndMin = 0 }
        end
		local nightMode = IsInRange(args.StartHour, args.StartMin, args.EndHour, args.EndMin)
		local delay  = 0
		if not nightMode == true then
			delay = SecondsTill(args.StartHour, args.StartMin)
		end
		return "auto", delay, "manual setting detected"
end
