function Evaluate(_, state, devices, _)
    if #devices == 0 then
        return state, 0, "no devices found"
    end
    local homeUsers = FilterDevices(devices, true)
    local wantHeating = #homeUsers > 0
    local reason = "all users are away: " .. ListDevices(devices)
    local delay = 900
    if wantHeating then
        reason = "one or more users are home: " .. ListDevices(homeUsers)
        delay = 0
    end
    if state.Heating == wantHeating then
        return state, 0, reason
    end
    return { Overlay = not wantHeating, Heating = wantHeating }, delay, reason
end

function FilterDevices(devices, state)
    local filtered = {}
    for _, device in ipairs(devices) do
        if device.Home == state then
            table.insert(filtered, device)
        end
    end
    return filtered
end

function ListDevices(devices)
    local result = ""
    for _, obj in ipairs(devices) do
        result = result .. obj.Name .. ", "
    end
    return result:sub(1, -3)
end
