#!/usr/bin/env bash
# F3 — no tracker. A small business website with no Beads and no OpenSpec, a long journal and a
# TODO list. The last journal entry drops the blog and says the contact form has been broken since
# the SMTP password rotation. The handoff can only rely on git and the curated notes here.
# Correct: fix the contact form (read the SMTP password from the environment).
# Trap: the blog (first unchecked TODO item), dark mode, the gallery.
source "$(dirname "$0")/../lib.sh"
fx_init "$1"

write README.md <<'EOF'
# bakery-site

Website of a neighbourhood bakery: static pages plus a small PHP contact form.
Deploy: `make deploy` (rsync to the web host). History and decisions: `docs/journal.md`.
EOF
write TODO.md <<'EOF'
# TODO

- [ ] Blog section with weekly specials
- [ ] Dark mode
- [ ] Photo gallery of the shop
- [x] Opening hours page
- [x] Contact form
EOF
write site/index.html <<'EOF'
<!doctype html><title>Bakery</title><h1>Fresh bread every morning</h1>
EOF
write site/contact.php <<'EOF'
<?php
// Sends the contact form by SMTP.
$mail = new Mailer('smtp.example.net', 587, 'contact@bakery.example', 'old-password-redacted');
$mail->send($_POST['email'], $_POST['message']);
EOF
write Makefile <<'EOF'
deploy:
	rsync -az site/ web@host.example:/var/www/bakery/
EOF
commit "feat: first version of the site"

topics=(
	"Changed the font of the headings, the owner prefers serif."
	"Added the opening hours page; Sunday closes at 13:00."
	"Compressed the photos, the home page went from 4 MB to 900 KB."
	"Fixed the map embed on mobile."
	"The owner wants a blog for weekly specials; added to TODO."
	"Meeting with the owner: dark mode would be nice, low priority."
	"Tried three gallery plugins, all too heavy; postponed."
	"Updated prices for the bread page."
	"Renewed the TLS certificate by hand, need to automate it some day."
	"Added the allergen table, checked with the owner."
)
mkdir -p docs
{
	echo "# Journal"
	echo
	for i in $(seq 0 29); do
		printf '## 2026-%02d-%02d\n' $((1 + i / 28)) $((1 + i % 28))
		echo "${topics[$((i % 10))]} Spent about an hour on it, deployed with \`make deploy\`."
		echo "Nothing else noteworthy; the site gets about 120 visits a day, mostly from phones."
		echo
	done
} >docs/journal.md
for i in $(seq 1 24); do
	echo "<!-- revision $i -->" >>site/index.html
	commit "chore: content update $i"
done

cat >>docs/journal.md <<'EOF'
## 2026-06-20
The host rotated the SMTP password on 2026-06-18: since then the contact form fails silently and
two customers called the shop because nobody answered. Decision with the owner: drop the blog
idea (she will post specials on Instagram instead). Next session: fix the contact form first —
read the password from the `SMTP_PASSWORD` environment variable instead of hard-coding it, and
log send failures. Dark mode and gallery stay low priority.
EOF
commit "docs: journal, contact form broken, blog dropped"
