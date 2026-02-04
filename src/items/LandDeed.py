from src.items.Item import Item, ItemRarity, ItemType

class LandDeed(Item):
    def __init__(self, name, price, desc, rarity):
        super().__init__(name, price, desc, rarity=rarity, item_type=ItemType.permanent)

    async def use(self, ctx, **kwargs):
        await ctx.send(f"📜 Ceci est l'acte de propriété pour **{self.name}**. Tape `!farm` pour t'y rendre.")
        return False


class VegetablePatchDeed(LandDeed):
    def __init__(self):
        super().__init__(
            "Terrain : Potager",
            500,
            "Un lopin de terre fertile pour faire pousser des légumes.",
            ItemRarity.common
        )

class GreenhouseDeed(LandDeed):
    def __init__(self):
        super().__init__(
            "Terrain : Serre Tropicale",
            1000,
            "Une structure en verre chauffée pour le café et le cacao.",
            ItemRarity.epic
        )

class OrchardDeed(LandDeed):
    def __init__(self):
        super().__init__(
            "Terrain : Verger Enchanté",
            10000,
            "Une île flottante magique. Seuls les fruits légendaires y poussent.",
            ItemRarity.legendary
        )