package datasetengine

var catalog = []ScenarioDefinition{
	{ID: ScenarioResidentArrival, Name: "Resident arrival", TaskIDs: allTasks(), Description: "A synthetic resident enters a home."},
	{ID: ScenarioResidentDeparture, Name: "Resident departure", TaskIDs: allTasks(), Description: "A synthetic resident exits a home."},
	{ID: ScenarioRoomMovement, Name: "Room-to-room movement", TaskIDs: allTasks(), Description: "An occupant moves along explicit room relations."},
	{ID: ScenarioMultipleOccupants, Name: "Multiple occupants", TaskIDs: allTasks(), Description: "Two or more synthetic occupants move independently."},
	{ID: ScenarioDoorOccupancy, Name: "Door and occupancy correlation", TaskIDs: allTasks(), Description: "Door and occupancy observations may be correlated or independent."},
	{ID: ScenarioLighting, Name: "Lighting transitions", TaskIDs: allTasks(), Description: "Lighting state changes through routine or manual causes."},
	{ID: ScenarioDailyRoutine, Name: "Daily routine", TaskIDs: allTasks(), Description: "A repeated but variable synthetic household routine."},
	{ID: ScenarioRoutineChange, Name: "Routine change", TaskIDs: allTasks(), HeldOut: true, Description: "A synthetic routine changes gradually or abruptly."},
	{ID: ScenarioTemporaryGuest, Name: "Temporary guest", TaskIDs: allTasks(), Description: "A guest visits for a finite synthetic interval."},
	{ID: ScenarioUnexpectedPresence, Name: "Unexpected presence", TaskIDs: allTasks(), HeldOut: true, Description: "Presence differs from the generated routine; it is not an intrusion label."},
	{ID: ScenarioSensorUnavailable, Name: "Sensor unavailable", TaskIDs: allTasks(), Description: "A source becomes unavailable while physical state continues."},
	{ID: ScenarioSensorRecovery, Name: "Sensor recovery", TaskIDs: allTasks(), Description: "A source resumes after an outage, possibly with delayed records."},
	{ID: ScenarioDelayedObservation, Name: "Delayed observations", TaskIDs: allTasks(), Description: "Network delivery occurs after the source observation."},
	{ID: ScenarioOutOfOrder, Name: "Out-of-order observations", TaskIDs: allTasks(), Description: "Source records arrive in a different order from generation."},
	{ID: ScenarioClockDrift, Name: "Clock drift", TaskIDs: allTasks(), HeldOut: true, Description: "Source timestamps have hidden fixed offset or drift."},
	{ID: ScenarioConflictingSources, Name: "Conflicting observations", TaskIDs: allTasks(), Description: "Sources report different values for one entity and attribute."},
	{ID: ScenarioMissingEvent, Name: "Missing event", TaskIDs: allTasks(), Description: "A source emission is omitted while other observations remain."},
	{ID: ScenarioNewDevice, Name: "New device", TaskIDs: allTasks(), HeldOut: true, Description: "A device appears after the training inventory was defined."},
	{ID: ScenarioModifiedTopology, Name: "Modified topology", TaskIDs: allTasks(), Description: "An explicit relation is added or changed after the cutoff."},
	{ID: ScenarioUnseenHomeTopology, Name: "Unseen home topology", TaskIDs: allTasks(), HeldOut: true, Description: "A topology family is held out from training groups."},
}

func ScenarioCatalog() []ScenarioDefinition {
	result := make([]ScenarioDefinition, len(catalog))
	for i, entry := range catalog {
		result[i] = entry
		result[i].TaskIDs = append([]TaskID(nil), entry.TaskIDs...)
	}
	return result
}

func allTasks() []TaskID {
	return []TaskID{TaskNextState, TaskTemporalTransition, TaskTopologyTransition, TaskBehaviorAnomaly, TaskObservationConsistency}
}
