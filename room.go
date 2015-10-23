package main

import (
	"fmt"
	"log"
	"time"

	calendar "google.golang.org/api/calendar/v3"
)

// the amount of time (in minutes) in which the room should be free
// in order for the room to be 'next available'
//
const minRoomAvailabilityPeriod = 15

type Room struct {
	Name   string
	Events []Event
}

type RoomSet struct {
	Rooms       map[string]Room
	TotalRooms  int
	RoomsLoaded int
}

func (r Room) Available() bool {
	if len(r.Events) != 0 {
		firstEvent := r.Events[0]
		if firstEvent.StartAt().Before(time.Now()) && firstEvent.EndAt().After(time.Now()) {
			return false
		}
	}
	return true
}

func (r Room) NextAvailable() time.Time {
	if len(r.Events) == 0 {
		return time.Now()
	}

	var prevEvent = r.Events[0]
	var minTimeBeforeNextEvent time.Time

	for _, currentEvent := range r.Events {
		minTimeBeforeNextEvent = prevEvent.EndAt().Add(minRoomAvailabilityPeriod * time.Minute)

		if minTimeBeforeNextEvent.Before(currentEvent.StartAt()) {
			return prevEvent.EndAt()
		}

		prevEvent = currentEvent
	}

	return time.Time{}
}

func (r Room) AvailableUntil() time.Time {
	if len(r.Events) == 0 {
		return time.Time{}
	}

	firstEvent := r.Events[0]
	if firstEvent.StartAt().Before(time.Now()) {
		return time.Time{}
	}

	return firstEvent.StartAt()
}

func CreateRoomFromEvents(calendarId, roomName string, calendarEvents []*calendar.Event) Room {
	room := Room{
		Name: roomName,
	}

	for _, calendarEvent := range calendarEvents {
		// filter the event if the room hasn't accepted the booking request
		if !roomAccepted(calendarId, roomName, calendarEvent) {
			continue
		}

		event := Event{
			Source: calendarEvent,
		}
		room.Events = append(room.Events, event)
	}
	return room
}

func roomAccepted(calendarId, roomName string, calendarEvent *calendar.Event) bool {
	room := filterAttendees(calendarId, calendarEvent.Attendees)
	if room != nil {
		return room.ResponseStatus == "accepted"
	}

	// TODO(jabley): handle a bunch of fun things:
	// - private events
	// - events booked directly in a room calendar (rather than inviting a room to an event)
	//
	if calendarEvent.Visibility == "private" {
		log.Printf("No visibility of private event %v in %v. Assuming it wins\n", calendarEvent.Id, roomName)
		return true
	}

	if len(calendarEvent.Attendees) == 0 {
		log.Printf("No attendees for event %v in %v – assuming it's booked direct to the room's calendar\n",
			calendarEvent.Summary,
			roomName)
		return true
	}

	log.Printf("Unable to find room %v in event %v\n", roomName, calendarEvent.Summary)
	attendeeStr := ""
	for _, attendee := range calendarEvent.Attendees {
		attendeeStr = attendeeStr + fmt.Sprintf("\t%v %v %v\n", attendee.DisplayName, attendee.Email, attendee.Id)
	}
	log.Printf("Attendees: %v\n", attendeeStr)

	return true
}

func filterAttendees(calendarId string, attendees []*calendar.EventAttendee) *calendar.EventAttendee {
	for _, attendee := range attendees {
		if attendee.Email == calendarId {
			return attendee
		}
	}
	return nil
}
