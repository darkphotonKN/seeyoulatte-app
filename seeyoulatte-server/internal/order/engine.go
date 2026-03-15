package order

import (
	"fmt"
	"log/slog"

	"github.com/darkphotonKN/seeyoulatte-app/internal/constants"
)

func TransitionOrder(transitionCxt TransitionContext, order *Order, targetState constants.OrderState, actor constants.Actor) error {
	for _, transition := range transitions {
		// validate state
		fromCheck := constants.OrderState(order.State) == transition.From
		toCheck := targetState == transition.To
		actorCheck := actor == transition.Actor

		if !fromCheck || !toCheck || !actorCheck {
			continue
		}

		// FOUND MATCHING CASE

		// matches the current transition states
		if transition.Guard != nil {
			err := transition.Guard(order, transitionCxt)

			if err != nil {
				slog.Warn("Error caught when guard called during state transition combinations in attempted transition.",
					"order_id", order.ID,
					"order_state", order.State,
					"target_state", targetState,
				)
				return fmt.Errorf("Error caught when guard called during state transition combinations in attempted transition.")
			}
		}

		// carry out side effects through actions
		if transition.Action != nil {
			err := transition.Action(order, transitionCxt)
			if err != nil {
				slog.Warn("Error caught when action called during state transition combinations in attempted transition.",
					"order_id", order.ID,
					"order_state", order.State,
					"target_state", targetState,
				)
				return fmt.Errorf("Error caught when action called during state transition combinations in attempted transition.")
			}
		}

		// make the state transition
		order.State = string(targetState)

		// found match, no more search needed
		return nil
	}

	return fmt.Errorf("No matching from and to state transition combinations in attempted transition.")
}
