function Evaluate(_, zone, _, args)
		if not zone.Overlay  then
			return zone, 0, "no manual setting detected"
		end
		zone.Overlay = false
        if args == nil or #args ~= 4 then
            args = { StartHour = 23, StartMin = 30, EndHour = 6, EndMin = 0 }
        end
		local delay  = 0
		if not IsInRange(args.StartHour, args.StartMin, args.EndHour, args.EndMin) then
			delay = SecondsTill(args.StartHour, args.StartMin)
		end
		return zone, delay, "manual setting detected"
end
