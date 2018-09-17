const combat = window.combat = window.combat || {};
const $app = document.querySelector('#app');

combat.createTag = (tagName, attrs) => {
	const tag = document.createElement(tagName);

	if (typeof attrs === 'object') {
		Object.keys(attrs).forEach(key => {
			const value = attrs[key];

			if (key !== 'children') {
				return value !== '' && tag.setAttribute(key, value);
			}

			if (typeof value === 'string' || typeof value === 'number') {
				tag.innerHTML = value;
			} else if (typeof value === 'object') {
				if (value.constructor === Array) {
					tag.append(...value);
				} else {
					tag.append(value);
				}
			}
		});
	}

	return tag;
}

if (typeof window.combatLogs === 'object') {
	// init after all
	setTimeout(() => combat.renderTable($app, window.combatLogs), 0);
}

// combat.$detailsPlaceholder = document.querySelector('#details');

// combat.openDetails = hashId => {
// 	const targetLogs = combat.logs[hashId];

// 	if (targetLogs.length !== 0) {
// 		combat.renderLogNavigation(targetLogs);
// 	}
// }