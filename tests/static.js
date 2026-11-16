import http from 'k6/http'
import { check } from 'k6'

const BASE = __ENV.BASE_URL || 'http://localhost:3000'
const JSON_HEADER = { 'Content-Type': 'application/json' }
const BASE_HEADER = { headers: JSON_HEADER }

function new_auth_header(token) {
    return { headers: { ...JSON_HEADER, Authorization: `Bearer ${token}` } }
}

// 1 iteration, fails the run if any check fails
export const options = {
    vus: 1,
    iterations: 1,
    thresholds: { checks: [ 'rate==1' ] },
}

// post /users; create a new user
function new_user() {
    const email = `user${Date.now()}${Math.random()}@test.dev`
    const creds = JSON.stringify({ email, password: 'password123' })

    check(http.post(`${BASE}/auth/register`, creds, BASE_HEADER),
        { 'register 201': (r) => r.status === 201 })
    check(http.post(`${BASE}/auth/register`, creds, BASE_HEADER),
        { 'duplicate register 409': (r) => r.status === 409 })

    const bad = JSON.stringify({ email, password: 'wrong-password' })
    check(http.post(`${BASE}/auth/login`, bad, BASE_HEADER),
        { 'wrong password 401': (r) => r.status === 401 })

    const res = http.post(`${BASE}/auth/login`, creds, BASE_HEADER)
    check(res,
        { 'login 200': (r) => r.status === 200 })

    return res.json().access_token
}

export default function() {

    const dummy_user = new_user()

    // GET /ready; confirm api is ready and has a connection to the database
    check(http.get(`${BASE}/ready`),
        { 'ready 200': (r) => r.status === 200 })

    // /auth/me; check if auth works
    check(http.get(`${BASE}/auth/me`),
        { 'me without token 401': (r) => r.status === 401 })
    check(http.get(`${BASE}/auth/me`, new_auth_header(dummy_user)),
        { 'me with token 200': (r) => r.status === 200 })

    // POST /event; create a new event
    const user_event_hoster = new_user()
    const user_event_hoster_header = new_auth_header(user_event_hoster)
    const at = new Date(Date.now() + 365 * 864e5)
    const body = JSON.stringify({
        name: 'Test',
        venue: 'Hall',
        starts_at: at.toISOString()
    })
    check(http.post(`${BASE}/event`, body, BASE_HEADER),
        { 'create event with no token 401': (r) => r.status === 401 })
    check(http.post(`${BASE}/event`, JSON.stringify({ name: '' }), user_event_hoster_header),
        { 'create event invalid 400': (r) => r.status === 400 })
    const created_event = http.post(`${BASE}/event`, body, user_event_hoster_header)
    check(created_event,
        { 'create event 201': (r) => r.status === 201 })
    const created_event_id = created_event.json().id
    http.post(`${BASE}/event`, body, user_event_hoster_header) // another event with same info

    // GET /event; get a list of events, specified by time
    const before = encodeURIComponent(new Date(at - 1000).toISOString())
    const page1 = http.get(`${BASE}/event?limit=1&after_starts_at=${before}`).json()
    check(page1,
        { 'page1 has 1': (p) => p.length === 1 })
    const cursor = encodeURIComponent(page1[0].starts_at)
    const page2 = http.get(`${BASE}/event?limit=1&after_starts_at=${cursor}&after_id=${page1[0].id}`).json()
    check(page2,
        { 'page2 not skipped (same starts_at)': (p) => p.length === 1 })

    // GET /event/{id}; get information of a specific event
    const created_event_info = http.get(`${BASE}/event/${created_event_id}`)
    check(created_event_info,
        { 'get event info 200': (r) => r.status === 200 })
    check(created_event_info.json(),
        { 'sections is array': (event_info) => Array.isArray(event_info.sections) })
    check(http.get(`${BASE}/event/not-a-event-uuid`),
        { 'bad get event info 400': (r) => r.status === 400 })
    check(http.get(`${BASE}/event/00000000-0000-0000-0000-000000000000`),
        { 'get missing event info 404': (r) => r.status === 404 })

    // POST /event/{id}/section; create a new section in a event that YOU created
    const section = JSON.stringify({ name: 'A', price: 5000, rows: 5, seats_per_row: 10 })
    const section_url = `${BASE}/event/${created_event_id}/section`
    check(http.post(section_url, section, user_event_hoster_header),
        { 'create section in event 201': (r) => r.status === 201 })
    check(http.post(section_url, section, user_event_hoster_header),
        { 'duplicate section 409': (r) => r.status === 409 })
    check(http.post(section_url, section, new_auth_header(dummy_user)),
        { 'section by non-organizer 403': (r) => r.status === 403 })
    check(http.post(section_url, JSON.stringify({ name: 'B' }), user_event_hoster_header),
        { 'section invalid 400': (r) => r.status === 400 })

    const seats = http.get(`${BASE}/event/${created_event_id}`).json().sections[0]
    check(seats,
        { '50 seats, all available': (s) => s.total_seats === 50 && s.available_seats === 50 })
}
