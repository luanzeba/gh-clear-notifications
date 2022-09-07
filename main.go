package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
)

type Subject struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

type Notification struct {
	Id              string
	Unread          bool
	Reason          string
	UpdatedAt       string
	LastReadAt      string
	Subject         Subject
	Url             string
	SubscriptionUrl string `json:"subscription_url"`
}

type NotificationList []Notification

type PullRequest struct {
	Url   string
	State string
	Title string
}

func (n NotificationList) Filter(f func(Notification) bool) NotificationList {
	var r NotificationList
	for _, v := range n {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}

func (n NotificationList) MarkAsReadAndUnsubscribe(client api.RESTClient, err error) {
	for _, v := range n {
		// Mark as read
		s, _ := json.Marshal([]interface{}{})
		b := bytes.NewBuffer(s)
		err = client.Patch(v.Url, b, nil)
		if err != nil {
			fmt.Println("Error marking as read:", err)
		}

		// Unsubscribe
		err = client.Delete(strings.Trim(v.SubscriptionUrl, "https://api.github.com"), nil)
		if err != nil {
			fmt.Println("Error unsubscribing:", err)
		}

		fmt.Printf("d")
	}
}

func main() {
	client, err := gh.RESTClient(nil)
	if err != nil {
		fmt.Println("Error creating client:", err)
		return
	}

	// // Used for debugging certain attributes
	// result := NotificationList{}
	// err = client.Get("notifications", &result)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// for _, n := range result {
	// 	fmt.Println("title:", n.Subject.Title)
	// 	fmt.Println("subscription_url:", strings.Trim(n.SubscriptionUrl, "https://api.github.com"))
	// }

	allNotifications := NotificationList{}
	allDeployNotifications := NotificationList{}
	allCiNotifications := NotificationList{}
	allReviewedPrNotifications := NotificationList{}
	i := 1

	for {
		paginatedNotificationList := NotificationList{}
		url := fmt.Sprintf("notifications?page=%v", i)
		err = client.Get(url, &paginatedNotificationList)
		if err != nil {
			fmt.Println("Error fetching notifications:", err)
		}
		if len(paginatedNotificationList) == 0 {
			break
		}
		i += 1

		allNotifications = append(allNotifications, paginatedNotificationList...)

		deployTrains := paginatedNotificationList.Filter(func(n Notification) bool {
			return n.Subject.Type == "PullRequest" && strings.Contains(n.Subject.Title, "Grouped deploy branch train")
		})
		allDeployNotifications = append(allDeployNotifications, deployTrains...)

		ci := paginatedNotificationList.Filter(func(n Notification) bool {
			return n.Reason == "ci_activity"
		})
		allCiNotifications = append(allCiNotifications, ci...)

		pullRequest := PullRequest{}
		reviewedPrNotifications := paginatedNotificationList.Filter(func(n Notification) bool {
			if n.Reason == "review_requested" {
				err = client.Get(strings.Trim(n.Subject.URL, "https://api.github.com"), &pullRequest)
				fmt.Printf("*")
				if err != nil {
					fmt.Println("Error fetching pull request:", err)
				}

				return pullRequest.State == "closed"
			}
			return false
		})
		allReviewedPrNotifications = append(allReviewedPrNotifications, reviewedPrNotifications...)

		fmt.Printf(".")
	}

	allDeployNotifications.MarkAsReadAndUnsubscribe(client, err)
	allCiNotifications.MarkAsReadAndUnsubscribe(client, err)
	allReviewedPrNotifications.MarkAsReadAndUnsubscribe(client, err)

	fmt.Println("Total notifications:", len(allNotifications))
	fmt.Println("Total deploy trains:", len(allDeployNotifications))
	fmt.Println("Total CI:", len(allCiNotifications))
	fmt.Println("Total reviewed PRs:", len(allReviewedPrNotifications))
}
