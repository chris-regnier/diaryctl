package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/spf13/cobra"
)

// seedTemplates defines reusable templates to create during seeding.
var seedTemplates = []storage.Template{
	{
		Name:    "daily",
		Content: "## Morning\n\n\n## Afternoon\n\n\n## Evening\n\n",
	},
	{
		Name:    "gratitude",
		Content: "## Grateful For\n\n1. \n2. \n3. \n\n## Highlight of the Day\n\n",
	},
	{
		Name:    "standup",
		Content: "## Yesterday\n\n\n## Today\n\n\n## Blockers\n\n",
	},
	{
		Name:    "weekly-review",
		Content: "## Wins This Week\n\n\n## Challenges\n\n\n## Next Week Focus\n\n",
	},
}

// profile defines a user persona for generating seed data.
type profile struct {
	name        string
	description string
	// daysBack is how far back to start generating entries.
	daysBack int
	// frequency is the approximate probability of writing on any given day (0.0–1.0).
	frequency float64
	// jotChance is the probability of adding jot-style sub-entries on days with entries.
	jotChance float64
	// templates lists template names this profile tends to use.
	templates []string
	// entries is a pool of content generators.
	entries []func(day time.Time, rng *rand.Rand) string
}

var profiles = map[string]profile{
	"daily-writer": {
		name:        "daily-writer",
		description: "Consistent daily journaler who rarely misses a day",
		daysBack:    90,
		frequency:   0.92,
		jotChance:   0.3,
		templates:   []string{"daily", "gratitude"},
		entries: []func(day time.Time, rng *rand.Rand) string{
			dailyMorningRoutine,
			dailyReflection,
			dailyGratitude,
			dailyFreeform,
			dailyProductivity,
		},
	},
	"weekend-journaler": {
		name:        "weekend-journaler",
		description: "Writes mostly on weekends and occasionally on weekdays",
		daysBack:    120,
		frequency:   0.0, // handled specially per weekday/weekend
		jotChance:   0.1,
		templates:   []string{"gratitude"},
		entries: []func(day time.Time, rng *rand.Rand) string{
			weekendAdventure,
			weekendReading,
			weekendCooking,
			weekendSocial,
			dailyFreeform,
		},
	},
	"dev-standup": {
		name:        "dev-standup",
		description: "Developer using diaryctl for work standups (weekdays only)",
		daysBack:    60,
		frequency:   0.0, // handled specially — weekdays only
		jotChance:   0.4,
		templates:   []string{"standup", "weekly-review"},
		entries: []func(day time.Time, rng *rand.Rand) string{
			devStandup,
			devDebugging,
			devFeatureWork,
			devCodeReview,
			devPlanning,
		},
	},
}

var seedCmd = &cobra.Command{
	Use:   "seed [profile]",
	Short: "Seed the diary with realistic sample data",
	Long: `Populate the diary with realistic entries to simulate an active user.

Available profiles:
  daily-writer      – Consistent daily journaler (~90 days, rarely misses)
  weekend-journaler – Writes mostly on weekends (~120 days)
  dev-standup       – Developer standups on weekdays (~60 days)

If no profile is specified, "daily-writer" is used.`,
	Example: `  diaryctl seed
  diaryctl seed daily-writer
  diaryctl seed weekend-journaler
  diaryctl seed dev-standup
  diaryctl seed --list`,
	Args:     cobra.MaximumNArgs(1),
	PostRunE: invalidateCachePostRun,
	RunE: func(cmd *cobra.Command, args []string) error {
		listProfiles, _ := cmd.Flags().GetBool("list")
		if listProfiles {
			fmt.Fprintln(os.Stdout, "Available profiles:")
			for name, p := range profiles {
				fmt.Fprintf(os.Stdout, "  %-20s %s\n", name, p.description)
			}
			return nil
		}

		profileName := "daily-writer"
		if len(args) > 0 {
			profileName = args[0]
		}

		p, ok := profiles[profileName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: unknown profile %q\n", profileName)
			fmt.Fprintln(os.Stderr, "Run 'diaryctl seed --list' to see available profiles.")
			os.Exit(1)
		}

		rng := rand.New(rand.NewSource(time.Now().UnixNano()))

		// Create templates used by this profile
		templatesCreated := 0
		for _, st := range seedTemplates {
			// Only create templates relevant to the profile (or all if profile uses them)
			relevant := false
			for _, tn := range p.templates {
				if tn == st.Name {
					relevant = true
					break
				}
			}
			if !relevant {
				continue
			}

			tid, err := entry.NewID()
			if err != nil {
				return fmt.Errorf("generating template ID: %w", err)
			}
			now := time.Now().UTC()
			t := storage.Template{
				ID:        tid,
				Name:      st.Name,
				Content:   st.Content,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := store.CreateTemplate(t); err != nil {
				// Template may already exist — skip silently
				continue
			}
			templatesCreated++
		}

		// Generate entries across the date range
		now := time.Now()
		startDate := now.AddDate(0, 0, -p.daysBack)
		entriesCreated := 0
		jotsCreated := 0

		for day := startDate; !day.After(now); day = day.AddDate(0, 0, 1) {
			if !shouldWrite(p, day, rng) {
				continue
			}

			// Pick a random content generator
			gen := p.entries[rng.Intn(len(p.entries))]
			content := gen(day, rng)

			// Determine template refs
			var templateRefs []entry.TemplateRef
			if len(p.templates) > 0 && rng.Float64() < 0.6 {
				tname := p.templates[rng.Intn(len(p.templates))]
				tmpl, err := store.GetTemplateByName(tname)
				if err == nil {
					templateRefs = []entry.TemplateRef{
						{TemplateID: tmpl.ID, TemplateName: tmpl.Name},
					}
				}
			}

			id, err := entry.NewID()
			if err != nil {
				return fmt.Errorf("generating entry ID: %w", err)
			}

			// Create entry at a realistic time of day
			entryTime := randomTimeOfDay(day, rng)
			e := entry.Entry{
				ID:        id,
				Content:   strings.TrimSpace(content),
				CreatedAt: entryTime.UTC(),
				UpdatedAt: entryTime.UTC(),
				Templates: templateRefs,
			}

			if err := store.Create(e); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: skipping entry for %s: %v\n", day.Format("2006-01-02"), err)
				continue
			}
			entriesCreated++

			// Maybe add jot-style follow-up entries
			if rng.Float64() < p.jotChance {
				jotCount := 1 + rng.Intn(3)
				for j := 0; j < jotCount; j++ {
					jotContent := randomJot(rng)
					jotID, err := entry.NewID()
					if err != nil {
						continue
					}
					jotTime := entryTime.Add(time.Duration(1+rng.Intn(8)) * time.Hour)
					if jotTime.After(now) {
						jotTime = now.Add(-time.Duration(rng.Intn(60)) * time.Minute)
					}
					je := entry.Entry{
						ID:        jotID,
						Content:   jotContent,
						CreatedAt: jotTime.UTC(),
						UpdatedAt: jotTime.UTC(),
					}
					if err := store.Create(je); err != nil {
						continue
					}
					jotsCreated++
				}
			}
		}

		if jsonOutput {
			fmt.Fprintf(os.Stdout, `{"profile":"%s","templates_created":%d,"entries_created":%d,"jots_created":%d}`+"\n",
				profileName, templatesCreated, entriesCreated, jotsCreated)
		} else {
			fmt.Fprintf(os.Stdout, "Seeded with profile %q:\n", profileName)
			fmt.Fprintf(os.Stdout, "  Templates created: %d\n", templatesCreated)
			fmt.Fprintf(os.Stdout, "  Entries created:   %d\n", entriesCreated)
			fmt.Fprintf(os.Stdout, "  Jots created:      %d\n", jotsCreated)
		}

		return nil
	},
}

func init() {
	seedCmd.Flags().Bool("list", false, "list available profiles")
	rootCmd.AddCommand(seedCmd)
}

// shouldWrite determines if this profile would write on the given day.
func shouldWrite(p profile, day time.Time, rng *rand.Rand) bool {
	wd := day.Weekday()
	switch p.name {
	case "weekend-journaler":
		if wd == time.Saturday || wd == time.Sunday {
			return rng.Float64() < 0.85
		}
		return rng.Float64() < 0.15
	case "dev-standup":
		if wd == time.Saturday || wd == time.Sunday {
			return false
		}
		return rng.Float64() < 0.88
	default:
		return rng.Float64() < p.frequency
	}
}

// randomTimeOfDay returns a time on the given day at a realistic hour.
func randomTimeOfDay(day time.Time, rng *rand.Rand) time.Time {
	// Most journal entries happen between 7am and 10pm
	hour := 7 + rng.Intn(15)
	minute := rng.Intn(60)
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}

// randomJot generates a quick timestamped jot entry.
func randomJot(rng *rand.Rand) string {
	jots := []string{
		"Quick thought: need to follow up on that email from earlier.",
		"Reminder: pick up groceries on the way home.",
		"Had a good conversation with a colleague about project architecture.",
		"Feeling energized after a short walk outside.",
		"Interesting article about distributed systems — save for later.",
		"Need to schedule dentist appointment this week.",
		"Coffee with an old friend was exactly what I needed today.",
		"That bug was caused by a race condition. Fixed with a mutex.",
		"Idea: what if we used event sourcing for the audit log?",
		"Finally figured out the CI pipeline issue. Was a Docker layer caching problem.",
		"Note to self: update the team wiki with the new deployment steps.",
		"Took a 20-minute power nap. Feeling much sharper now.",
		"Weather is beautiful — should plan a weekend hike.",
		"Read an interesting thread about Go error handling patterns.",
		"Shipped the feature flag implementation. Clean rollout so far.",
	}
	return jots[rng.Intn(len(jots))]
}

// --- Content generators ---

func dailyMorningRoutine(day time.Time, rng *rand.Rand) string {
	mornings := []string{
		"Woke up at 6:30, went for a run along the river. The fog was still lifting — beautiful morning.",
		"Started the day with meditation and a long breakfast. Feeling centered.",
		"Up early to catch the sunrise. Made pour-over coffee and read for an hour before work.",
		"Rough night's sleep but pushed through a morning workout. The cold shower afterward helped.",
		"Lazy morning. Stayed in bed reading until 9. Sometimes you need that.",
	}
	afternoons := []string{
		"Productive afternoon — got through most of my task list. Focused deeply on the main project for about 3 hours.",
		"Meetings took up most of the afternoon. Need to protect my calendar better.",
		"Spent the afternoon at the library. Changed my environment, changed my output.",
		"Afternoon slump hit hard. Took a walk and came back to tackle the remaining items.",
		"Had lunch with a friend I haven't seen in months. Good to reconnect.",
	}
	evenings := []string{
		"Quiet evening at home. Cooked pasta, watched a documentary about deep-sea exploration.",
		"Went to a local jazz show. The trumpet player was incredible.",
		"Evening run, then journaling. The consistency is starting to feel natural.",
		"Video call with family. My niece learned to ride a bike — she was so proud.",
		"Read before bed. Currently halfway through a book on systems thinking.",
	}
	return fmt.Sprintf("## Morning\n\n%s\n\n## Afternoon\n\n%s\n\n## Evening\n\n%s",
		mornings[rng.Intn(len(mornings))],
		afternoons[rng.Intn(len(afternoons))],
		evenings[rng.Intn(len(evenings))])
}

func dailyReflection(day time.Time, rng *rand.Rand) string {
	reflections := []string{
		"Today I noticed I'm more patient than I used to be. Small interactions that used to frustrate me — waiting in line, slow responses — don't bother me anymore. Growth is quiet sometimes.",
		"Spent some time thinking about where I want to be in a year. Not in terms of achievements, but in terms of how I want to feel day-to-day. Calm, purposeful, connected.",
		"Had a difficult conversation today. Said what I needed to say, even though it was uncomfortable. Proud of myself for not avoiding it.",
		"The project at work is hitting a rough patch. Instead of stressing, I mapped out what I can control and let go of the rest. It helped.",
		"Realized I've been neglecting some friendships. Sent a few messages to people I haven't talked to in a while. Already got two warm responses back.",
		"Journaling consistently has made me more aware of patterns in my mood. I notice now when I'm heading into a slump and can adjust before it hits.",
		"Read something today that stuck with me: \"The days are long but the years are short.\" Trying to hold that perspective.",
		"Failed at something today, and it stung. But I'm trying to see it as data rather than judgment. What can I learn from this?",
	}
	return reflections[rng.Intn(len(reflections))]
}

func dailyGratitude(day time.Time, rng *rand.Rand) string {
	items := []string{
		"A warm cup of coffee on a cold morning",
		"The sound of rain on the window",
		"A kind word from a stranger",
		"Having meaningful work to do",
		"A good night's sleep",
		"The smell of fresh bread from the bakery down the street",
		"A long phone call with an old friend",
		"Finding a quiet spot in the park to read",
		"The way sunlight comes through the kitchen window in the afternoon",
		"A perfectly ripe avocado",
		"My health, which I too often take for granted",
		"The local library — an incredible free resource",
		"A colleague who helped me debug a tricky issue",
		"The walk home through the neighborhood",
		"Music that matches exactly how I'm feeling",
		"Having a comfortable bed to sleep in",
		"Clean water from the tap",
		"The satisfaction of crossing things off a list",
	}
	rng.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })

	highlight := []string{
		"Finished a book I'd been working through for weeks. That feeling of completion.",
		"Had an unexpectedly deep conversation over lunch.",
		"Solved a problem I'd been stuck on for days. The answer was simpler than I thought.",
		"Caught a gorgeous sunset on the walk home.",
		"Received positive feedback on a project I put a lot of effort into.",
		"Tried a new recipe and it actually turned out great.",
	}

	return fmt.Sprintf("## Grateful For\n\n1. %s\n2. %s\n3. %s\n\n## Highlight of the Day\n\n%s",
		items[0], items[1], items[2],
		highlight[rng.Intn(len(highlight))])
}

func dailyFreeform(day time.Time, rng *rand.Rand) string {
	entries := []string{
		"Not sure what to write today. Some days are just... days. Went to work, came home, made dinner. The ordinariness of it is its own kind of comfort, I think. Not every day needs to be remarkable.",
		"Tried a new coffee shop on the east side. The espresso was excellent — better than my usual place. The barista recommended a single-origin Ethiopian that was fruity and bright. Might become a regular.",
		"Long walk through the botanical gardens after work. The dahlias are in full bloom. Took some photos but they don't capture it. You need to stand there and breathe it in.",
		"Spent the evening reorganizing my bookshelf. Found three books I forgot I owned. Started re-reading one of them — it hits different the second time around.",
		"Rainy day. The kind where you don't want to go outside but the sound of it is perfect for focusing. Got a lot of reading done. Made soup for dinner — the kitchen smelled amazing all evening.",
		"Met up with the running group this morning. We did 8k along the canal. The pace felt easy, which is a nice change from struggling through every run a few months ago. Consistency pays off.",
		"Cooked for friends tonight. Made a Thai curry from scratch — ground the paste by hand and everything. Everyone went for seconds, which is the best compliment.",
		"Spent the day at a workshop on creative writing. Didn't expect to enjoy it so much. The exercises pushed me to write in styles I'd never try on my own. Might sign up for the full course.",
	}
	return entries[rng.Intn(len(entries))]
}

func dailyProductivity(day time.Time, rng *rand.Rand) string {
	entries := []string{
		"Knocked out five tasks before lunch. The trick was starting with the hardest one — once that was done, everything else felt easy. Also blocked my calendar for two hours of deep work in the afternoon and it was the most productive stretch of the week.",
		"Today I tried time-boxing: 25 minutes on, 5 off. It worked surprisingly well for the tedious admin stuff. Managed to clear my entire inbox and organize next week's schedule.",
		"Slow start but picked up after lunch. Wrote the proposal draft, reviewed two PRs, and outlined the Q2 roadmap. Not bad for a day that started with zero motivation.",
		"Focused on one thing all day: the migration plan. No context-switching, no meetings. By end of day I had a complete document with rollback procedures. Should do this more often.",
		"Experimented with working from the café instead of home. The ambient noise was perfect. Finished the design doc and sketched out three implementation approaches. Sometimes a change of scenery is all you need.",
	}
	return entries[rng.Intn(len(entries))]
}

func weekendAdventure(day time.Time, rng *rand.Rand) string {
	adventures := []string{
		"Hiked to the summit today. The trail was steeper than expected but the view at the top made it worthwhile. Could see the city skyline through the haze. Packed sandwiches and ate lunch on a flat rock overlooking the valley. Legs are going to be sore tomorrow.",
		"Day trip to the coast. The drive was scenic — rolling hills and farmland giving way to cliffs and ocean. Walked along the beach for an hour collecting shells. Found a tide pool full of anemones and tiny crabs. Fish and chips at a harbor-side pub to close it out.",
		"Explored a neighborhood I'd never been to. Found an incredible used bookstore — floor to ceiling, organized by some system only the owner understands. Walked out with four books and a recommendation for a nearby ramen shop. The ramen was outstanding.",
		"Rented kayaks and paddled around the lake. The water was perfectly still in the morning — like glass. Saw a heron standing in the shallows, completely unbothered by us. Packed it in after three hours and had ice cream on the dock.",
		"Visited the farmers' market early. Bought fresh peaches, sourdough bread, and a jar of local honey. Spent the rest of the morning wandering through side streets, ducking into small galleries and antique shops. A good aimless morning.",
		"Cycled the riverside trail — about 40km round trip. Stopped at a beer garden at the halfway point for a cold drink and watched boats go by. The ride back was into a headwind so it was a proper workout.",
	}
	return adventures[rng.Intn(len(adventures))]
}

func weekendReading(day time.Time, rng *rand.Rand) string {
	entries := []string{
		"Spent most of the day reading on the porch. Finished the novel I started last week. The ending was unexpected — bittersweet in a way that felt earned. Immediately started the next book on my list, a collection of essays about solitude and creativity.",
		"Rain all day, which was perfect. Made a pot of tea and worked through several chapters of a book on the history of mathematics. The chapter on the development of zero was fascinating — how a concept of nothing became everything in math.",
		"Went to the library and browsed for an hour. Picked up a biography, a poetry collection, and a science book about mycelium networks. The mycelium book is incredible — forests communicate through underground fungal networks. Nature is endlessly surprising.",
		"Lazy reading day. Re-read some favorite short stories. There's something comforting about returning to writing you know well. You notice new things every time — a sentence you glossed over before suddenly resonates.",
	}
	return entries[rng.Intn(len(entries))]
}

func weekendCooking(day time.Time, rng *rand.Rand) string {
	entries := []string{
		"Big cooking day. Made a batch of pasta sauce from scratch — San Marzano tomatoes, garlic, basil from the windowsill. Also baked sourdough for the first time in months. The crumb was better than expected. Fed the starter, which I'd been neglecting.",
		"Tried making croissants from scratch. The lamination process took all morning — roll, fold, chill, repeat. The result wasn't bakery-perfect but they were flaky, buttery, and honestly delicious. Will try again next weekend.",
		"Meal prepped for the week. Roasted a whole chicken, made three different grain bowls, and a big batch of miso soup. The kitchen was a disaster by the end but I love having food ready to go on busy weekdays.",
		"Hosted a small dinner party. Made risotto as the main — mushroom and thyme. The key is patience with the stock, ladleful by ladleful. Everyone lingered at the table for hours after, talking and finishing the wine. Those are the best evenings.",
	}
	return entries[rng.Intn(len(entries))]
}

func weekendSocial(day time.Time, rng *rand.Rand) string {
	entries := []string{
		"Brunch with the group today. We've been doing this monthly and it's become something I really look forward to. Good food, easy conversation, lots of laughing. Walked around the park together afterward. Simple and perfect.",
		"Game night at a friend's place. Played a strategy board game that took three hours. I lost spectacularly but it was incredibly fun. The host made homemade pizza and we stayed up way too late.",
		"Visited my parents for the afternoon. Mom made her famous soup and Dad showed me his garden — the tomatoes are coming in strong this year. These visits remind me to slow down.",
		"Coffee with a friend I hadn't seen in months. We talked for two hours without checking our phones once. She's going through some changes and it was good to just listen. Genuine connection is rare and valuable.",
		"Potluck at the community center. Met several neighbors I'd never spoken to. A retired teacher told stories about the neighborhood from 30 years ago. There's so much history in the people around us.",
	}
	return entries[rng.Intn(len(entries))]
}

func devStandup(day time.Time, rng *rand.Rand) string {
	yesterdays := []string{
		"Finished the API endpoint for user preferences. Added validation and wrote integration tests. Also reviewed two PRs — one for the search feature and one for the logging refactor.",
		"Spent most of the day on the database migration. Had to handle the edge case where existing records have null timestamps. Wrote a backfill script and tested it against a snapshot of production data.",
		"Pair-programmed with Alex on the caching layer. We settled on a write-through strategy with TTL-based invalidation. Got the core implementation done and passing tests.",
		"Investigated the memory leak reported by the ops team. Tracked it down to a goroutine that wasn't being cleaned up on connection close. Fix was small but the detective work took a while.",
		"Set up the new CI pipeline with GitHub Actions. Parallel test execution cut our build time from 12 minutes to 4. Also added a linting step that caught three dormant issues.",
	}
	todays := []string{
		"Planning to tackle the notification service refactor. Current implementation is tightly coupled to email — need to abstract it so we can add Slack and webhook channels.",
		"Will finish the pagination PR and address the review comments. Then starting on the data export feature — need to support CSV and JSON formats.",
		"Focus today is on writing tests for the auth middleware. Coverage is at 65% and we want to get it above 80% before the release.",
		"Going to set up structured logging across the service layer. Using zerolog — need to define our log levels and context fields as a team standard.",
		"Working on the WebSocket implementation for real-time updates. Need to handle reconnection logic and message ordering. Will start with a prototype.",
	}
	blockers := []string{
		"None today.",
		"Waiting on the design team for the notification preferences mockup.",
		"Need access to the staging database — ticket is pending with infrastructure.",
		"Blocked on the API spec for the third-party integration. Following up with the partner team.",
		"No blockers, but the flaky test in the payment module keeps wasting CI minutes. Should prioritize fixing it.",
	}
	return fmt.Sprintf("## Yesterday\n\n%s\n\n## Today\n\n%s\n\n## Blockers\n\n%s",
		yesterdays[rng.Intn(len(yesterdays))],
		todays[rng.Intn(len(todays))],
		blockers[rng.Intn(len(blockers))])
}

func devDebugging(day time.Time, rng *rand.Rand) string {
	entries := []string{
		"Spent the morning debugging a nasty race condition in the job queue. The issue only manifested under load — two workers would occasionally pick up the same job. Root cause was a missing `SELECT ... FOR UPDATE` in the claim query. Added the lock and wrote a concurrent test to verify.",
		"Tracked down why the API was returning 500s intermittently. Turned out the connection pool was exhausted because we weren't closing response bodies in one of the HTTP client calls. Classic Go gotcha — `defer resp.Body.Close()` saves lives.",
		"Investigated a data inconsistency reported by the support team. Found that our event handler was processing messages out of order during high throughput. Added sequence number validation and a small reorder buffer. Wrote a chaos test that injects random delays to verify the fix.",
		"Memory profiling session today. The service was using 3x more memory than expected. pprof showed a massive allocation in our JSON serialization path — we were creating new encoders for every request. Switched to a sync.Pool and memory dropped by 60%.",
		"The frontend team reported that search was slow. Added query EXPLAIN to the slow endpoint and found a missing index on the `tags` column. After adding a GIN index, query time went from 800ms to 12ms. Sometimes the fix is simple.",
	}
	return entries[rng.Intn(len(entries))]
}

func devFeatureWork(day time.Time, rng *rand.Rand) string {
	entries := []string{
		"Started the file upload feature today. Designed the API contract — multipart upload with progress tracking. Using pre-signed URLs for direct-to-S3 uploads to keep the server stateless. Wrote the handler skeleton and the storage interface. Will wire up the frontend tomorrow.",
		"Implemented the role-based access control system. Three roles: viewer, editor, admin. Permissions are checked at the middleware level using a policy engine pattern. Wrote a comprehensive test matrix covering all role × resource × action combinations. 47 test cases, all green.",
		"Built out the webhook delivery system. Entries go into a queue, a worker picks them up, attempts delivery with exponential backoff (max 5 retries over 24 hours). Dead-lettered webhooks are logged and surfaced in the admin dashboard. The retry logic was the trickiest part — had to handle partial failures gracefully.",
		"Finished the data export pipeline. Users can request an export of their data, which runs async and produces a ZIP file with JSON and attachments. Used a temp directory pattern with cleanup on completion. Added rate limiting — one export per user per hour — to prevent abuse.",
		"Spent the day on the notification preferences feature. Users can now choose per-channel (email, in-app, mobile push) and per-event (mentions, replies, status changes) notification settings. The UI was straightforward but the backend needed a denormalized lookup table for efficient delivery-time queries.",
	}
	return entries[rng.Intn(len(entries))]
}

func devCodeReview(day time.Time, rng *rand.Rand) string {
	entries := []string{
		"Reviewed a large PR for the billing service migration. The code was well-structured but I flagged a few things: missing error wrapping, an N+1 query in the invoice generation loop, and a potential nil pointer in the discount calculation. Good discussion in the comments — the author agreed and pushed fixes within an hour.",
		"Did a thorough review of the new GraphQL resolvers. Suggested splitting the monolithic resolver file into domain-specific modules. Also caught a potential data leak — the resolver was exposing internal IDs that should have been opaque. Sensitive stuff in API design.",
		"Reviewed the infrastructure-as-code PR for the new staging environment. Mostly Terraform with some Helm charts. Flagged that the database instance size was too small for realistic load testing and that the secret management was using env vars instead of the vault integration.",
		"Two smaller reviews today. One was a dependency update PR — checked the changelogs for breaking changes and ran the test suite locally. The other was a bug fix for timezone handling in the reporting module. The fix was correct but I suggested adding a regression test with the specific timezone that triggered the bug.",
		"Reviewed the authentication refactor. The move from session-based to JWT was clean, but I had concerns about token revocation. We discussed it and agreed on a hybrid approach: short-lived JWTs (15 min) with a refresh token stored server-side. Added this to the architecture decision record.",
	}
	return entries[rng.Intn(len(entries))]
}

func devPlanning(day time.Time, rng *rand.Rand) string {
	entries := []string{
		"Sprint planning today. We committed to 34 story points — ambitious but achievable. Main deliverables: complete the search overhaul, ship the mobile notification improvements, and start the API versioning project. I volunteered to lead the API versioning effort since I've been thinking about it for a while.",
		"Roadmap review with the team leads. Q2 is going to be heavy on infrastructure: multi-region deployment, improved observability, and the database migration from Postgres to CockroachDB for the high-write tables. I'm excited but also slightly terrified about the database migration.",
		"Architecture decision session for the event system redesign. Evaluated three options: polling, webhooks, and server-sent events. Went with SSE for the real-time feed and webhooks for external integrations. Documented the decision in our ADR format with pros/cons for each option.",
		"Backlog grooming session. Went through 20 tickets, estimated 15, and sent 5 back for more requirements. The product team appreciated the pushback — better to clarify now than discover ambiguity mid-sprint. Also identified three tech debt items to prioritize.",
		"One-on-one with my manager about career growth. We discussed the path to senior engineer — it's less about writing more code and more about multiplying the team's impact. She suggested I mentor the two junior devs on the team and lead the next cross-team initiative.",
	}
	return entries[rng.Intn(len(entries))]
}
