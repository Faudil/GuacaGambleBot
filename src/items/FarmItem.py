from src.items.Item import ItemRarity
from src.items.ResourceItem import ResourceItem


class RottenPlant(ResourceItem):
    def __init__(self):
        super().__init__(
            "Plante Pourrie", 0, "Tu as mal géré ta ferme...", ItemRarity.common,
            {"stat": "max_hp", "amount": 0}
        )

class Wheat(ResourceItem):
    def __init__(self):
        super().__init__(
            "Blé", 5, "Indispensable pour faire du pain.", ItemRarity.common,
            {"stat": "max_hp", "amount": 2}
        )

class Oat(ResourceItem):
    def __init__(self):
        super().__init__(
            "Avoine", 8, "Parfait pour le petit déjeuner.", ItemRarity.common,
            {"stat": "speed", "amount": 2}
        )

class Corn(ResourceItem):
    def __init__(self):
        super().__init__(
            "Maïs", 12, "Fait aussi du pop-corn !", ItemRarity.common,
            {"stat": "max_hp", "amount": 2}
        )

class Potato(ResourceItem):
    def __init__(self):
        super().__init__(
            "Patate", 20, "On peut en faire de la vodka...", ItemRarity.rare,
            {"stat": "defense", "amount": 2}
        )

class Tomato(ResourceItem):
    def __init__(self):
        super().__init__(
            "Tomate", 25, "Un fruit ou un légume ? Le débat continue.", ItemRarity.rare,
            {"stat": "max_hp", "amount": 3}
        )

class Pumpkin(ResourceItem):
    def __init__(self):
        super().__init__(
            "Citrouille", 40, "Parfait pour Halloween.", ItemRarity.rare,
            {"stat": "defense", "amount": 2}
        )

class CoffeeBean(ResourceItem):
    def __init__(self):
        super().__init__(
            "Grain de Café", 60, "L'or noir du matin.", ItemRarity.epic,
            {"stat": "speed", "amount": 2}
        )

class CocoaBean(ResourceItem):
    def __init__(self):
        super().__init__(
            "Fève de Cacao", 75, "L'ingrédient principal du bonheur (chocolat).", ItemRarity.epic,
            {"stat": "acc", "amount": 2}
        )

class Strawberry(ResourceItem):
    def __init__(self):
        super().__init__(
            "Fraise", 90, "Rouge, sucrée et juteuse.", ItemRarity.epic,
            {"stat": "crit_c", "amount": 2}
        )

class GoldenApple(ResourceItem):
    def __init__(self):
        super().__init__(
            "Pomme Dorée", 150, "Elle brille d'une lueur magique.", ItemRarity.epic,
            {"stat": "max_hp", "amount": 5}
        )

class StarFruit(ResourceItem):
    def __init__(self):
        super().__init__(
            "Fruit Étoile", 250, "Un fruit cosmique d'une autre dimension.", ItemRarity.legendary,
            {"stat": "crit_d", "amount": 0.2}
        )

# --- SEEDS ---

class Seed(ResourceItem):
    def __init__(self, name, price, desc, crop_class, grow_time, rarity=ItemRarity.common):
        super().__init__(name, price, desc, rarity=rarity)
        self.crop_class = crop_class
        self.grow_time = grow_time

    async def use(self, ctx, **kwargs):
        from src.database.settings import get_language
        from src.utils.i18n import t
        lang = get_language(ctx.guild.id if ctx.guild else None)
        await ctx.send(t("farm.seed_use_hint", lang, seed=self.display_name(lang)))
        return False

class WheatSeed(Seed):
    def __init__(self):
        super().__init__("Graine de Blé", 2, "À planter pour obtenir du blé (5 min).", Wheat, 300)

class OatSeed(Seed):
    def __init__(self):
        super().__init__("Graine d'Avoine", 3, "À planter pour obtenir de l'avoine (10 min).", Oat, 600)

class CornSeed(Seed):
    def __init__(self):
        super().__init__("Graine de Maïs", 5, "À planter pour obtenir du maïs (30 min).", Corn, 1800)

class PotatoSeed(Seed):
    def __init__(self):
        super().__init__("Graine de Patate", 8, "À planter pour obtenir des patates (1h).", Potato, 3600, ItemRarity.rare)

class TomatoSeed(Seed):
    def __init__(self):
        super().__init__("Graine de Tomate", 10, "À planter pour obtenir des tomates (2h).", Tomato, 7200, ItemRarity.rare)

class PumpkinSeed(Seed):
    def __init__(self):
        super().__init__("Graine de Citrouille", 15, "À planter pour obtenir des citrouilles (4h).", Pumpkin, 14400, ItemRarity.rare)

class CoffeeSeed(Seed):
    def __init__(self):
        super().__init__("Graine de Café", 25, "À planter pour obtenir du café (8h).", CoffeeBean, 28800, ItemRarity.epic)

class CocoaSeed(Seed):
    def __init__(self):
        super().__init__("Graine de Cacao", 30, "À planter pour obtenir du cacao (12h).", CocoaBean, 43200, ItemRarity.epic)

class StrawberrySeed(Seed):
    def __init__(self):
        super().__init__("Graine de Fraise", 40, "À planter pour obtenir des fraises (18h).", Strawberry, 64800, ItemRarity.epic)

class GoldenAppleSeed(Seed):
    def __init__(self):
        super().__init__("Pépin de Pomme Dorée", 75, "À planter pour obtenir des pommes dorées (24h).", GoldenApple, 86400, ItemRarity.epic)

class StarFruitSeed(Seed):
    def __init__(self):
        super().__init__("Pépin de Fruit Étoile", 125, "À planter pour obtenir des fruits étoiles (48h).", StarFruit, 172800, ItemRarity.legendary)