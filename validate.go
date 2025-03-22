package goverter

import (
	"fmt"
)

func validateMethods(lookup *methodIndex[generatedMethod]) error {
	for _, hits := range lookup.Exact {
		for _, entry := range hits {
			genMethod := entry.Item

			if genMethod.Explicit && len(genMethod.RawFieldSettings) > 0 {
				isTargetStructPointer := genMethod.Target.Pointer && genMethod.parameters.Target.PointerInner.Struct
				if !genMethod.Target.Struct && !isTargetStructPointer {
					return fmt.Errorf("Invalid struct field mapping on method:\n    %s\n    %s\n\nField mappings like goverter:map or goverter:ignore may only be set on struct or struct pointers.\nSee https://goverter.jmattheis.de/guide/configure-nested", genMethod.Location, genMethod.ID)
				}
			}
		}
	}
	return nil
}
