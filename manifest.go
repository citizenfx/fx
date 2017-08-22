package main

import (
	"log"
	"path/filepath"

	"github.com/Shopify/go-lua"
)

// A Manifest represents the manifest of a resource.
type Manifest struct {
	values map[string][]string
}

// Get gets a metadata string from the manifest.
func (m *Manifest) Get(key string) string {
	if len(m.values[key]) == 0 {
		return ""
	}

	return m.values[key][0]
}

// GetAll gets all metadata strings from the manifest.
func (m *Manifest) GetAll(key string) []string {
	return m.values[key]
}

func openManifest(resourceRoot string) (*Manifest, error) {
	resourceInitCode := `
return function(chunk)
    local addMetaData = AddMetaData

    setmetatable(_G, {
    	__index = function(t, k)
    		local raw = rawget(t, k)

    		if raw then
    			return raw
    		end

    		return function(value)
    			local newK = k

    			if type(value) == 'table' then
    				-- remove any 's' at the end (client_scripts, ...)
    				if k:sub(-1) == 's' then
    					newK = k:sub(1, -2)
    				end

    				-- add metadata for each table entry
    				for _, v in ipairs(value) do
    					addMetaData(newK, v)
    				end
    			else
    				addMetaData(k, value)
    			end

    			-- for compatibility with legacy things
    			return function(v2)
					if type(v2) == 'string' then
						addMetaData(k .. '_extra', v2)
					end
    			end
    		end
    	end
    })

    -- execute the chunk
    chunk()

    -- and reset the metatable
    setmetatable(_G, nil)
end
	`

	l := lua.NewState()
	lua.OpenLibraries(l)

	// remove any unsafe libraries
	list := []string{"ffi", "require", "dofile", "load", "loadfile", "package", "os", "io"}

	for _, pkg := range list {
		l.PushNil()
		l.SetGlobal(pkg)
	}

	manifest := &Manifest{}
	manifest.values = map[string][]string{}

	// add AddMetaData function
	l.PushGoFunction(func(state *lua.State) int {
		key, keyOk := l.ToString(1)
		value, valueOk := l.ToString(2)

		if keyOk && valueOk {
			manifest.values[key] = append(manifest.values[key], value)
		}

		return 0
	})

	l.SetGlobal("AddMetaData")

	if err := lua.LoadString(l, resourceInitCode); err != nil {
		log.Printf("couldn't load string: %v", err)
		return nil, err
	}

	if err := l.ProtectedCall(0, 1, 0); err != nil {
		log.Printf("couldn't call core function: %v", err)
		return nil, err
	}

	if err := lua.LoadFile(l, filepath.Join(resourceRoot, "__resource.lua"), ""); err != nil {
		return nil, err
	}

	if err := l.ProtectedCall(1, 0, 0); err != nil {
		log.Printf("couldn't call function: %v", err)
		return nil, err
	}

	return manifest, nil
}
