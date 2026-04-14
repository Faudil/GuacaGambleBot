class Recipe:
    def __init__(self, result_item: str, ingredients: dict[str, int], level_required: int, xp_reward: int):
        self.result_item = result_item.lower()
        self.ingredients = {k.lower(): v for k, v in ingredients.items()}
        self.level_required = level_required
        self.xp_reward = xp_reward

RECIPES: dict[str, Recipe] = {
    "bière": Recipe("bière", {"blé": 3}, 1, 10),
    "café": Recipe("café", {"grain de café": 3}, 1, 10),
    "ticket à gratter": Recipe("ticket à gratter", {"charbon": 1, "caillou": 1}, 1, 10),
    "engrais": Recipe("engrais", {"plante pourrie": 3, "charbon": 1}, 2, 15),
    "potion d'oubli": Recipe("potion d'oubli", {"plante pourrie": 2, "poisson-globe": 1}, 2, 20),
    "fortune cookie": Recipe("fortune cookie", {"blé": 2, "fraise": 1}, 2, 20),
    "arc": Recipe("arc", {"avoine": 2, "caillou": 2}, 3, 25),
    "aimant rouillé": Recipe("aimant rouillé", {"minerai de fer": 3, "caillou": 5}, 3, 20),
    "hameçon": Recipe("hameçon", {"minerai de fer": 1, "minerai d'argent": 1}, 3, 25),
    "parchemin d'identité": Recipe("parchemin d'identité", {"plante pourrie": 2, "minerai d'argent": 1}, 4, 35),
    "aimant": Recipe("aimant", {"minerai de fer": 5, "minerai de cuivre": 1}, 5, 40),
    "pièce truquée": Recipe("pièce truquée", {"pépite d'or": 1, "caillou": 2, "charbon": 1}, 5, 45),
    "jeton de casino": Recipe("jeton de casino", {"pépite d'or": 1, "minerai d'argent": 1}, 6, 50),
    "terrain : potager": Recipe("terrain : potager", {"pépite d'or": 2, "caillou": 20}, 7, 80),
    "aimant électrique": Recipe("aimant électrique", {"platine": 2, "minerai de cuivre": 5}, 7, 60),
    "terrain : serre tropicale": Recipe("terrain : serre tropicale", {"pépite d'or": 5, "platine": 2}, 9, 120),
    "ticket vip": Recipe("ticket vip", {"diamant brut": 3, "platine": 2}, 9, 150),
    "terrain : verger enchanté": Recipe("terrain : verger enchanté", {"diamant brut": 2, "émeraude": 2}, 10, 250),
    "œuf mystère": Recipe("œuf mystère", {"diamant brut": 1, "pomme dorée": 1, "adn pur": 1, "poussière d'os": 10}, 10, 200),
}
