package plan

import "fmt"

func Validate(p Plan) error {
	seenIDs := map[OperationID]struct{}{}
	seenCreates := map[string]struct{}{}
	opsByResource := map[string][]Operation{}
	createNetworkIDs := map[OperationID]struct{}{}
	deleteContainerIDs := map[OperationID]struct{}{}

	for _, op := range p.Operations {
		if _, ok := seenIDs[op.ID]; ok {
			return fmt.Errorf("duplicate operation id: %s", op.ID)
		}
		seenIDs[op.ID] = struct{}{}

		resourceKey := string(op.Resource.Kind) + "/" + op.Resource.Name
		opsByResource[resourceKey] = append(opsByResource[resourceKey], op)

		if op.Kind == OpCreateContainer {
			if _, ok := seenCreates[resourceKey]; ok {
				return fmt.Errorf("duplicate create target: %s", resourceKey)
			}
			seenCreates[resourceKey] = struct{}{}
		}
		if op.Kind == OpCreateNetwork {
			createNetworkIDs[op.ID] = struct{}{}
		}
		if op.Kind == OpDeleteContainer {
			deleteContainerIDs[op.ID] = struct{}{}
		}
	}

	if err := validateIncompatiblePairs(opsByResource); err != nil {
		return err
	}
	if err := validateCreateContainerDependencies(p, createNetworkIDs); err != nil {
		return err
	}
	if err := validateDeleteNetworkDependencies(p, deleteContainerIDs); err != nil {
		return err
	}
	return nil
}

func validateCreateContainerDependencies(p Plan, createNetworkIDs map[OperationID]struct{}) error {
	if len(createNetworkIDs) == 0 {
		return nil
	}
	for _, op := range p.Operations {
		if op.Kind != OpCreateContainer {
			continue
		}
		depends := false
		for _, dep := range op.DependsOn {
			if _, ok := createNetworkIDs[dep]; ok {
				depends = true
				break
			}
		}
		if !depends {
			return fmt.Errorf("create_container %s must depend on create_network when network is absent", op.Resource.Name)
		}
	}
	return nil
}

func validateDeleteNetworkDependencies(p Plan, deleteContainerIDs map[OperationID]struct{}) error {
	if len(deleteContainerIDs) == 0 {
		return nil
	}
	for _, op := range p.Operations {
		if op.Kind != OpDeleteNetwork {
			continue
		}
		for id := range deleteContainerIDs {
			found := false
			for _, dep := range op.DependsOn {
				if dep == id {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("delete_network %s must depend on all delete_container operations", op.Resource.Name)
			}
		}
	}
	return nil
}

func validateIncompatiblePairs(opsByResource map[string][]Operation) error {
	for key, ops := range opsByResource {
		seen := map[OperationKind]struct{}{}
		for _, op := range ops {
			for existing := range seen {
				if incompatibleOnSameResource(existing, op.Kind) {
					return fmt.Errorf("incompatible operations on same resource %s: %s and %s", key, existing, op.Kind)
				}
			}
			seen[op.Kind] = struct{}{}
		}
	}
	return nil
}

func incompatibleOnSameResource(a, b OperationKind) bool {
	if a == b {
		return false
	}
	if (a == OpDeleteContainer && (b == OpCreateContainer || b == OpStartContainer || b == OpStopContainer || b == OpRecreateContainer)) ||
		(b == OpDeleteContainer && (a == OpCreateContainer || a == OpStartContainer || a == OpStopContainer || a == OpRecreateContainer)) {
		return true
	}
	if (a == OpDeleteNetwork && b == OpCreateNetwork) || (a == OpCreateNetwork && b == OpDeleteNetwork) {
		return true
	}
	if (a == OpCreateContainer && b == OpRecreateContainer) || (a == OpRecreateContainer && b == OpCreateContainer) {
		return true
	}
	return false
}
